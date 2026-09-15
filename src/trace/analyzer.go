// analyzer.go — the real engine behind `context-trace`: builds the
// same Data (see types.go) no matter which of the four doors asked
// for it.
package trace

import (
	"fmt"
	"os"
	"path/filepath"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/core/focus"
	focusrender "mova.local/core/focus/render"
	"mova.local/core/focus/astfilter"
	"mova.local/core/focus/resolvers"
	"mova.local/i18n"
	"mova.local/sanitize"
)

// componentOrder is the fixed display order for the context breakdown
var componentOrder = []string{"Agents", "Skills", "Prompt", "Focus", "Memory"}

// AnalyzeLocal analyzes an already-configured local project (with a project.json).
func AnalyzeLocal(adapter core.Adapter, root, projectName, taskName, origin string) (*Data, error) {
	proj, err := adapter.GetProject(projectName)
	if err != nil {
		return nil, err
	}
	resolvedTask := core.ResolveTaskName(proj, taskName)
	task, ok := proj.Tasks[resolvedTask]
	if !ok {
		return nil, fmt.Errorf("task %q does not exist in project %q", taskName, projectName)
	}

	report, err := budget.BuildReport(adapter, root, projectName, taskName, false)
	if err != nil {
		return nil, err
	}

	items := core.ResolveFocus(proj, &task)
	var focusCounts FocusCounts
	if len(items) > 0 {
		_, stats := focusrender.RenderFocusContext(root, proj.Repo, items, nil, core.ResolveExclude(proj, &task))
		focusCounts.Included = stats.FilesIncluded
		for _, n := range stats.ExcludedByDir {
			focusCounts.Excluded += n
		}
	}

	cfg := core.ResolveBudget(proj, &task)
	firewall := FirewallStatus{
		SanitizerOn:      core.SanitizerEnabled(cfg),
		PIIMaskingOn:     core.PIIMaskingEnabled(cfg),
		CacheGuardOn:     core.CacheGuardEnabled(cfg),
		CircuitBreakerOn: core.CircuitBreakerEnabled(cfg),
	}
	if report.PIIStats.TokensMasked > 0 {
		firewall.PIIWarning = fmt.Sprintf("%d possible sensitive-data fragments masked", report.PIIStats.TokensMasked)
	}

	d := &Data{
		Origin:          origin,
		ProjectName:     projectName,
		ProjectJSONPath: filepath.ToSlash(core.ProjectJSONPath(root, projectName)),
		TaskName:        resolvedTask,
		HasProjectJSON:  true,
		Focus:           focusCounts,
		Components:      componentsFromReport(report),
		Firewall:        firewall,
		TotalTokens:     report.TotalTokens,
		MaxTokens:       report.MaxTokens,
		Costs:           costRowsFromModelCosts(report.TotalCosts),
		Encoding:        report.Encoding,
		Report:          report,
		// Local project mode already runs its own mature Context Governance
		// pipeline (see firewall above and budget.BuildReport). The
		// per-file governance breakdown only runs in AnalyzeRemote's
		// discovery path - wiring both into one shared breakdown is a
		// documented follow-up (see docs/i18n/en/COMMANDS.md).
		GovernanceStatus: "CONTROLLED (project.json Context Governance)",
		PolicySource:     "project.json + config/policy.json (Context Governance)",
		ExecutionID:      NewExecutionID(),
		CommitHash:       CommitHashFor(proj.Repo),
	}
	return d, nil
}

// estimateImageTokens calcula los tokens de visión aproximados basándose en las dimensiones de la imagen.

// AnalyzeRemote analiza directorios sin project.json. root is the
// Mova Context install root (used only to load the governance policy
// cascade — config/policy.json / config/policy/*.json — via
// newGovernanceEngine; see governance.go).
func AnalyzeRemote(repoDir, repoURL, branch, origin, root, task string, ignorePatterns []string, prices *budget.PricesConfig, pruneDocstrings bool, onProgress ProgressFunc) (*Data, error) {
	info, err := os.Stat(repoDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("the directory to analyze does not exist or is not valid: %s", repoDir)
	}

	ctx := focus.Context{RepoPath: repoDir, Stats: &focus.ScanStats{}}
	var walked []string
	resolvers.WalkAllFiles(ctx, repoDir, func(path string) { walked = append(walked, path) })
	ctx.Stats.FilesIncluded = len(walked)

	focusCounts := FocusCounts{
		Included:           ctx.Stats.FilesIncluded,
		ExcludedSuggestion: suggestedExcludeDirs(ctx.Stats.ExcludedByDir),
	}
	for _, n := range ctx.Stats.ExcludedByDir {
		focusCounts.Excluded += n
	}

	g := newGovernanceEngine(root)

	// --ignore runs HERE, at Discovery time, before any file content
	// is read - an ignored file costs zero tokenization work and never
	// enters CANDIDATE, BM25 scoring, or cost estimation.
	var paths []string
	for _, path := range walked {
		if len(ignorePatterns) > 0 && MatchesIgnorePattern(RelativizePath(repoDir, path), ignorePatterns) {
			g.addExcludedReason(RelativizePath(repoDir, path), 0, i18n.T("reports.excluded_by_ignore_pattern"))
			continue
		}
		paths = append(paths, path)
	}

	totalFiles := len(paths)
	dirTokens := map[string]int{}
	dirFiles := map[string]int{}
	totalTokens := 0
	encoding := ""

	// Read every file's content ONCE, up front - both the security
	// evaluation loop below AND the optional task-relevance ranking
	// (see relevance.go) need it, and re-reading from disk twice for
	// every file would double this command's I/O for no reason.
	type readFile struct {
		path        string
		content     string
		imageTokens int
		readErr     error
	}
	reads := make([]readFile, len(paths))
	textByPath := map[string]string{}
	for i, path := range paths {
		content, imageTokens, err := getFileContentForTokenization(path)
		reads[i] = readFile{path: path, content: content, imageTokens: imageTokens, readErr: err}
		if err == nil && imageTokens == 0 && content != "" {
			textByPath[path] = content
		}
	}

	// Task-relevance narrowing (see relevance.go's RankByTask): only
	// runs when a task was actually given - with no task, CANDIDATE
	// keeps meaning exactly what the docs already disclose ("every
	// DISCOVERED file that passed focus/ignore"), never a silent,
	// unexplainable cut.
	var relevanceScores map[string]RelevanceScore
	notRelevant := map[string]bool{}
	if task != "" {
		relevanceScores, _ = RankByTask(textByPath, task, 0.35)
		notRelevant = SelectBelowCutoff(relevanceScores, 0.35)
	}
	prunedFiles := 0

	for i, r := range reads {
		path := r.path
		if r.readErr != nil {
			g.addExcluded(path, 0)
			continue
		}

		if notRelevant[path] {
			n, _, err := budget.CountTokens(r.content, "")
			if err == nil {
				g.recordDiscovered(n)
				g.addExcludedReason(RelativizePath(repoDir, path), n, i18n.T("reports.exclusion_reasons.not_relevant_to_task"))
			} else {
				g.addExcluded(RelativizePath(repoDir, path), 0)
			}
			continue
		}

		n := 0
		var enc string // Variable declarada en el scope del loop
		blocked := false
		relPath := RelativizePath(repoDir, path)
		switch {
		case r.imageTokens > 0:
			// Es una imagen: asignamos la estimación de vision tokens.
			// Las imágenes no pasan por el detector de secretos/PII de
			// texto — se registran directamente como ALLOWED.
			n = r.imageTokens
			if encoding == "" {
				encoding = "cl100k_base"
			}
			g.recordDiscovered(n)
			g.counts.CandidateFiles++
			g.counts.CandidateTokens += n
			g.counts.AllowedFiles++
			g.counts.AllowedTokens += n
		case r.content != "" && sanitize.IsAssetPath(path):
			// Binary/vector asset (SVG, images, fonts, sourcemaps): NO
			// semantic PII/secret scan by default - see sanitize.IsAssetPath's
			// doc comment for why (the "jina2.svg" false-positive case:
			// entropy-based detectors trip constantly on path-data /
			// embedded base64 that carries zero actual sensitive content).
			var err error
			n, enc, err = budget.CountTokens(r.content, "")
			if err != nil {
				g.addExcluded(relPath, 0)
				continue
			}
			encoding = enc
			g.recordDiscovered(n)
			g.counts.CandidateFiles++
			g.counts.CandidateTokens += n
			g.counts.AllowedFiles++
			g.counts.AllowedTokens += n
		case r.content != "":
			// Es texto: calculamos los tokens BPE
			var err error
			content := r.content
			if pruneDocstrings {
				if stripped, ok := astfilter.PruneDocs([]byte(content), path); ok {
					content = stripped
					prunedFiles++
				}
			}
			n, enc, err = budget.CountTokens(content, "")
			if err != nil {
				g.addExcluded(relPath, 0)
				continue
			}
			encoding = enc

			g.recordDiscovered(n)
			_, state := g.evaluateCandidate(relPath, r.content, n)
			blocked = state == StateBlocked
		default:
			g.addExcluded(relPath, 0)
			continue
		}

		if blocked {
			// A BLOCKED file never contributes to the final sendable
			// context — its tokens are already accounted for in
			// g.counts.BlockedTokens (see governance.go).
			continue
		}

		totalTokens += n

		dir := topLevelDir(repoDir, path)
		dirTokens[dir] += n
		dirFiles[dir]++

		if onProgress != nil && (i%40 == 0 || i == totalFiles-1) {
			pct := 20 + int(float64(i+1)/float64(totalFiles)*60)
			onProgress(pct, i18n.T("cli.messages.tokenizing", map[string]any{"done": i + 1, "total": totalFiles}))
		}
	}

	firewall := FirewallStatus{
		SanitizerOn:      false,
		PIIMaskingOn:     true,
		CacheGuardOn:     false,
		CircuitBreakerOn: false,
	}
	// Use FilesWithAnyIndicator (deduped, one count per file) — NOT
	// FilesWithPotentialPII+FilesWithPotentialSecrets, which double-counts
	// a file that has both a PII and a secret pattern and would then show
	// a DIFFERENT number here than the GOVERNANCE box's "Files with an
	// indicator" line and the audit JSON's files_with_any_indicator,
	// even though all three are meant to be the same metric (see the
	// "files_with_indicators vs files_would_sanitize" fix).
	if g.impact.FilesWithAnyIndicator > 0 {
		firewall.PIIWarning = fmt.Sprintf("%d file(s) with sensitive-data indicators (see files_with_any_indicator vs files_would_sanitize in pii-audit-log.json)", g.impact.FilesWithAnyIndicator)
	}

	finalTokens := g.counts.FinalSendableTokens()
	safeNowTokens := g.counts.SafeToSendNowTokens()

	d := &Data{
		Origin:           origin,
		IsRemote:         true,
		RepoURL:          repoURL,
		Branch:           branch,
		RepoDir:          repoDir,
		Focus:            focusCounts,
		Firewall:         firewall,
		TotalTokens:      totalTokens,
		MaxTokens:        0,
		Costs:            costRowsFromModelCosts(budget.EstimateCost(finalTokens, prices)),
		CostsSafeNow:     costRowsFromModelCosts(budget.EstimateCost(safeNowTokens, prices)),
		Encoding:         encoding,
		DirBreakdown:     buildDirBreakdown(dirTokens, dirFiles, totalTokens),
		GovernanceStatus: governanceStatus(false, g.counts),
		PolicySource:     g.policy.Source,
		PolicyVersion:    g.policy.Version,
		StateTotals:      g.counts,
		Findings:         g.findings,
		SecurityImpact:   g.impact,
		ExclusionReasons: g.exclusionRows(),
		ModelCompat:      modelCompatRows(prices, finalTokens, root),
		ExecutionID:      NewExecutionID(),
		CommitHash:       CommitHashFor(repoDir),
		TaskName:         task,
		IgnorePatterns:   ignorePatterns,
		PrunedDocstringFiles: prunedFiles,
		RelevanceTop:     topRelevanceRows(relevanceScores, textByPath, repoDir, 10),
	}

	return d, nil
}

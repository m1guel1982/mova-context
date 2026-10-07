// run_evidence.go — builds and writes the immutable evidence of one
// context release (mova.local/evidence) from a GatedContext. Every door
// that releases context calls RecordRun exactly once: `mova run`,
// get_full_context, chat_completion and `mova chat` (session start).
package budget

import (
	"path/filepath"
	"strings"

	"mova.local/core"
	"mova.local/evidence"
	"mova.local/mpaths"
)

// RunInfo is what only the door knows: where the context goes and who
// is on the other side.
type RunInfo struct {
	Door  string        // see evidence.Manifest.Door
	Agent evidence.Attr // e.g. observed:mcp-initialize, or "cli"
	Model evidence.Attr // the model that receives it, with its provenance
}

// RunsDir is projects/<project>/runs.
func RunsDir(root, project string) string {
	return filepath.Join(mpaths.ProjectsDir(root), project, "runs")
}

type runSpec struct {
	Focus            []string                  `json:"focus"`
	Exclude          []string                  `json:"exclude"`
	ReadScope        string                    `json:"read_scope"`
	DependencyPolicy map[string]string         `json:"dependency_policy"`
	AcceptMissing    []core.AcceptedDependency `json:"accept_missing,omitempty"`
	MaxTokens        int                       `json:"max_tokens,omitempty"`
	PIIMasking       bool                      `json:"pii_masking"`
}

type runGovernance struct {
	SanitizerLinesCollapsed  int   `json:"sanitizer_lines_collapsed"`
	SanitizerCommentsRemoved int   `json:"sanitizer_comments_removed"`
	PIIMasked                int   `json:"pii_masked"`
	Blocks                   []any `json:"changed_blocks,omitempty"`
}

// BaseManifest fills project, repo, config hashes, spec and author — the
// parts every run shares (also used for session runs).
func BaseManifest(root, project, task string, proj *core.Project, info RunInfo) evidence.Manifest {
	m := evidence.Manifest{Door: info.Door, Project: project, Task: task, Agent: info.Agent, Model: info.Model}
	if m.Agent.Value == "" {
		m.Agent = evidence.Attr{Value: "unknown", Source: "not_observable"}
	}
	if m.Model.Value == "" {
		m.Model = evidence.Attr{Value: "unknown", Source: "not_observable"}
	}
	m.Author = evidence.Attr{Value: core.ResolvePolicyAuthor(root, project), Source: "declared:project.json author / config/policy.json / MOVA_POLICY_AUTHOR (no verificado)"}
	if proj == nil {
		return m
	}
	m.Repo = evidence.GitState(core.RepoDir(root, proj))
	req := core.ResolvePolicyRequest(proj, core.OrchestratorPolicySelector(root), nil, nil)
	m.Config = append(m.Config, evidence.HashFile(core.ProjectJSONPath(root, project)))
	for _, p := range core.ResolvedPolicyFiles(root, req) {
		m.Config = append(m.Config, evidence.HashFile(p))
	}
	opt := GovernanceOptions(root, proj, nil)
	m.Policy = opt.Policy.Source

	sp := runSpec{
		Focus:            core.FocusInScope(proj, task),
		Exclude:          core.ExcludeInScope(proj, task),
		ReadScope:        core.ReadScopeFor(proj, task),
		DependencyPolicy: map[string]string{},
	}
	names := []string{task}
	if core.IsAllTasks(task) {
		names = core.SortedTaskNames(proj)
	}
	for _, n := range names {
		t := proj.Tasks[n]
		sp.DependencyPolicy[n] = core.DependencyPolicyFor(proj, &t)
		sp.AcceptMissing = append(sp.AcceptMissing, t.AcceptMissing...)
	}
	t := ResolveTask(proj, core.ResolveTaskName(proj, task))
	cfg := core.ResolveBudget(proj, t)
	if cfg != nil {
		sp.MaxTokens = cfg.MaxTokens
	}
	sp.PIIMasking = core.PIIMaskingEnabled(cfg)
	m.Spec = evidence.Raw(sp)
	return m
}

// RecordRun writes runs/<run_id>/ for one gated context. dryRun marks a
// context that passed every gate but was held back by egress_audit.
func RecordRun(root, project, task string, proj *core.Project, g GatedContext, info RunInfo, dryRun bool) (*evidence.Run, error) {
	scope := task
	if proj != nil && !core.IsAllTasks(core.NormalizeTaskArg(proj, task)) {
		scope = core.ResolveTaskName(proj, task)
	} else if proj != nil {
		scope = core.TaskAll
	}
	m := BaseManifest(root, project, scope, proj, info)

	if g.Sections != nil {
		for _, it := range g.Sections.FocusItems {
			m.Selection = append(m.Selection, it.Name)
		}
	}
	if g.Closure != nil {
		m.Closure = evidence.Raw(g.Closure)
	} else if core.ClosureHook == nil {
		m.Closure = evidence.Raw(map[string]string{"note": "validación de cierre no disponible en este binario"})
	}
	gov := runGovernance{
		SanitizerLinesCollapsed:  g.Sanitize.LinesRemoved,
		SanitizerCommentsRemoved: g.Sanitize.CommentsRemoved,
		PIIMasked:                g.PII.TokensMasked,
	}
	for _, b := range g.Governance {
		if b.Changed() {
			gov.Blocks = append(gov.Blocks, b)
		}
	}
	m.Govern = evidence.Raw(gov)
	m.Context.TokensEstimated = g.Tokens
	m.Context.Tokenizer = "tiktoken-go cl100k_base (estimación local; otros proveedores tokenizan distinto)"

	switch {
	case g.Err != nil:
		m.Decision = evidence.Decision{Outcome: "blocked", Gate: orDefault(g.Gate, "build"), Reason: firstLines(g.Err.Error(), 12)}
	case dryRun:
		m.Decision = evidence.Decision{Outcome: "dry_run", Reason: "egress_audit.dry_run: el contexto pasó todas las puertas pero no se liberó"}
	default:
		m.Decision = evidence.Decision{Outcome: "released"}
	}
	text := g.Text
	if g.Err != nil {
		text = ""
	}
	return evidence.Start(RunsDir(root, project), m, []byte(text))
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = append(lines[:n], "…")
	}
	return strings.Join(lines, "\n")
}

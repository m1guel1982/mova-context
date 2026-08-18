// build.go — BuildDiagram(adapter, root, name, task, detail, origin) is
// the ONE function that turns a project name (ordinary project OR
// multiagent group — see mova.local/orchestrator's package doc) into a
// diagram.Data value. It never re-implements project resolution,
// budget/Token-Firewall state, or token/cost counting: it reads
// core.Project fields directly for structure (agents/skills/jobs/focus)
// and delegates every NUMBER (tokens, costs, sanitizer/PII stats) to
// orchestrator.Count — the exact same call `mova run --count`, chat's
// "/budget", and the MCP "estimate_budget" tool already make. This is
// what "no inventar datos, leer el estado real del proyecto" (see the
// diagram feature's own spec) means in code: one source of truth, read
// twice for two different presentations (a number in a report, a box
// in a picture), never computed twice.
package diagram

import (
	"fmt"
	"strconv"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/models"
	"mova.local/orchestrator"
)

// compilerStages are the Context Compiler steps every project goes
// through, in order — always the same list (the engine has no
// per-project variation here), shown only when detail is verbose and
// the project actually has something for that stage to act on (e.g.
// "Focus" is only drawn if Focus is non-empty).
var compilerStages = []string{"Agents", "Skills", "Prompt", "Memory", "Focus"}

// ValidOrigins are the four doors a diagram render can be attributed
// to — see Data.Origin's own doc comment. Exported so callers (and
// tests) can validate against the same list this package uses
// internally, instead of hardcoding the four strings a second time.
var ValidOrigins = []string{"CLI", "Chat", "MCP", "API HTTP"}

// BuildDiagram reads project name (or group name) and returns the
// data its SVG/PNG/PDF render from. detail overrides project.json's
// own "diagram.detail_level" when non-empty (CLI --diagram always wins
// over the project's own default — see cli/diagram_cmd.go). origin
// identifies which of ValidOrigins triggered this render; an
// unrecognized/empty value falls back to "MCP" (see Data.Origin).
func BuildDiagram(adapter core.Adapter, root, name, task, detail, origin string) (*Data, error) {
	isGroup := orchestrator.IsGroup(root, name)
	d := &Data{
		ProjectName: name,
		IsGroup:     isGroup,
		Compiler:    compilerStages,
		Origin:      normalizeOrigin(origin),
		Interfaces:  []string{"CLI", "Chat", "MCP", "API HTTP"},
	}

	if isGroup {
		cfg, err := orchestrator.LoadGroupConfig(root, name)
		if err != nil {
			return nil, err
		}
		d.Description = cfg.Description
		seenSource := map[string]bool{}
		for _, agentName := range cfg.Agents {
			proj, err := adapter.GetProject(name + "/" + agentName)
			if err != nil {
				continue // a misconfigured agent doesn't stop the whole diagram
			}
			d.Agents = append(d.Agents, buildAgentNode(root, agentName, proj))
			for _, s := range sourcesFrom(proj) {
				if !seenSource[s.Path] {
					seenSource[s.Path] = true
					d.Sources = append(d.Sources, s)
				}
			}
			d.Jobs = append(d.Jobs, jobsFrom(proj)...)
			if d.Firewall == (Firewall{}) {
				d.Firewall = firewallFrom(proj) // representative — Token Firewall config is per-project, shown once from the first agent that sets one
			}
		}
	} else {
		proj, err := adapter.GetProject(name)
		if err != nil {
			return nil, err
		}
		d.Description = proj.Description
		d.Lang = proj.Lang
		d.Agents = []AgentNode{buildAgentNode(root, name, proj)}
		d.Sources = sourcesFrom(proj)
		d.Jobs = jobsFrom(proj)
		d.Firewall = firewallFrom(proj)
	}

	d.DetailLevel = resolveDetailLevel(detail, adapter, root, name, isGroup)

	// withFocus=true also computes the Budget & Focus comparison layer
	// (#18) — the same extra pass `mova budget --focus` already does,
	// reused here instead of a second implementation.
	if result, err := orchestrator.Count(adapter, root, name, task, true); err == nil {
		d.Metrics = metricsFrom(result, d.Agents)
	}
	return d, nil
}

// normalizeOrigin maps any case/spacing variant a caller might pass
// (CLI code, chat code, MCP arguments from an external client, HTTP
// JSON) onto the exact ValidOrigins string the SVG renderer expects —
// falling back to "MCP" for anything unrecognized, since raw MCP
// stdio/tool calls are the one door with no more specific signal to
// go on (see mcp/diagram_tool.go).
func normalizeOrigin(origin string) string {
	norm := strings.ToLower(strings.TrimSpace(origin))
	for _, valid := range ValidOrigins {
		if strings.ToLower(valid) == norm {
			return valid
		}
	}
	switch norm {
	case "http", "api", "http_api", "http-api":
		return "API HTTP"
	case "chat", "mova chat":
		return "Chat"
	case "cli", "mova run", "run":
		return "CLI"
	default:
		return "MCP"
	}
}

// resolveDetailLevel: CLI flag > project.json's own "diagram.detail_level" > verbose default.
// Reads project.json again only when detail=="" and name isn't a group
// (a group has no diagram config of its own — each agent could declare
// one, but the flag/default already covers that ambiguity).
func resolveDetailLevel(detail string, adapter core.Adapter, root, name string, isGroup bool) DetailLevel {
	if detail == string(DetailSimple) {
		return DetailSimple
	}
	if detail == string(DetailVerbose) {
		return DetailVerbose
	}
	if !isGroup {
		if proj, err := adapter.GetProject(name); err == nil && proj.Diagram != nil && proj.Diagram.DetailLevel == string(DetailSimple) {
			return DetailSimple
		}
	}
	return DetailVerbose
}

func buildAgentNode(root string, displayName string, proj *core.Project) AgentNode {
	n := AgentNode{
		Name:        displayName,
		Description: proj.Description,
		AgentRoles:  proj.Agents.Use,
		Skills:      proj.Skills.Use,
		PIIMasking:  core.PIIMaskingEnabled(proj.Budget), // THIS agent's own config — see model.go's doc comment on why this differs from Data.Firewall.PIIMaskingOn
	}
	for taskName := range proj.Tasks {
		n.Tasks = append(n.Tasks, taskName)
	}
	if proj.LLMProfile != nil {
		n.Provider = proj.LLMProfile.Provider
		n.ModelConfig = proj.LLMProfile.Config
		n.ModelName = n.ModelConfig // fallback until/unless resolveModel finds the real tag below
		if n.Provider == "" && n.ModelConfig != "" {
			// Same auto-resolution `mova run`/chat already apply (see
			// LLMProfile.Provider's own doc comment) — the diagram
			// should show the provider a real run would actually use,
			// not leave it blank just because project.json omitted the
			// disambiguation field.
			if resolved, err := models.ResolveConfigProvider(root, n.ModelConfig); err == nil {
				n.Provider = resolved
			}
		}
		resolveModel(root, &n)
	}
	return n
}

// resolveModel does the full traceability chain #11 asks for:
// project.json's llm_profile.{provider,config} -> the actual file at
// config/models/<provider>/<config>.json -> that file's own "model"
// (the real tag sent to the API, e.g. "llama3.2:3b") and "type" (which
// decides local vs. cloud — see core.LLMProfile.IsLocal's own doc
// comment on why that project.json field is NOT trustworthy for this:
// most projects never set it, while config/models/<provider>/<config>
// .json's "type" is what mova.local/models.NewProvider actually
// switches on to build the request, i.e. the one place this can't
// silently disagree with what a real run does).
func resolveModel(root string, n *AgentNode) {
	if n.Provider == "" || n.ModelConfig == "" {
		return
	}
	mc, err := models.DefaultCache.GetModel(root, n.Provider, n.ModelConfig)
	if err != nil {
		return // config file missing/unreadable — keep the ModelConfig fallback already set above, don't guess
	}
	if mc.ModelName != "" {
		n.ModelName = mc.ModelName
	}
	n.IsLocal = !isCloudType(mc.Type)
}

// isCloudType mirrors mova.local/models.NewProvider's own switch on
// ModelConfig.Type exactly (see provider.go) — every type value that
// routes to a cloud provider there is cloud here too; everything else
// (including the empty/default "ollama" case) is local.
func isCloudType(t string) bool {
	switch t {
	case "google", "gemini", "openai-compatible", "openai", "anthropic", "claude":
		return true
	default:
		return false
	}
}

// sourcesFrom collects Focus from the project's own top-level "focus"
// AND from every task's own Focus override (core.Task.Focus) — a task
// with its own focus is exactly as real a Source as the project-level
// one (see core.ResolveFocus, which BuildGatedContext already prefers
// per-task focus over project-level when both exist). Deduplicated so
// a path declared at both levels is only drawn once.
func sourcesFrom(proj *core.Project) []SourceRef {
	seen := map[string]bool{}
	var out []SourceRef
	add := func(f string) {
		if !seen[f] {
			seen[f] = true
			out = append(out, SourceRef{Path: f, Kind: kindOfFocus(f)})
		}
	}
	for _, f := range proj.Focus {
		add(f)
	}
	for _, t := range proj.Tasks {
		for _, f := range t.Focus {
			add(f)
		}
	}
	return out
}

func kindOfFocus(f string) string {
	switch {
	case len(f) > 0 && f[len(f)-1] == '/':
		return "dir"
	case containsGlob(f):
		return "glob"
	default:
		return "file"
	}
}

func containsGlob(s string) bool {
	for _, r := range s {
		if r == '*' || r == '?' {
			return true
		}
	}
	return false
}

func jobsFrom(proj *core.Project) []JobNode {
	out := make([]JobNode, 0, len(proj.Jobs))
	for _, j := range proj.Jobs {
		out = append(out, JobNode{
			Schedule:      j.Schedule,
			ScheduleHuman: humanizeCron(j.Schedule),
			Tasks:         j.Tasks,
			Save:          j.Save,
		})
	}
	return out
}

// humanizeCron turns the handful of cron patterns project.json's own
// job scheduler (mova.local/jobs) actually supports into plain text —
// only for patterns it can translate with full confidence; anything
// else (an unusual step value, a weekday LIST, "L"/"#" extensions,
// ...) is returned completely unchanged rather than risk showing a
// WRONG human-readable time. A person can always read the raw cron
// expression; showing a mistranslated one would be worse than not
// translating at all.
func humanizeCron(cron string) string {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return cron
	}
	min, hour, day, month, weekday := fields[0], fields[1], fields[2], fields[3], fields[4]

	if month != "*" {
		return cron // monthly/yearly-with-specific-month patterns: not worth the risk of a wrong translation
	}

	m, mErr := strconv.Atoi(min)
	h, hErr := strconv.Atoi(hour)

	switch {
	case min == "*" && hour == "*" && day == "*" && weekday == "*":
		return "Every minute"
	case mErr == nil && hour == "*" && day == "*" && weekday == "*":
		return fmt.Sprintf("Every hour, at minute %d", m)
	case mErr == nil && hErr == nil && day == "*" && weekday == "*":
		return fmt.Sprintf("Daily at %02d:%02d", h, m)
	case mErr == nil && hErr == nil && day == "*" && isSingleWeekday(weekday):
		return fmt.Sprintf("Weekly on %s at %02d:%02d", weekdayName(weekday), h, m)
	case mErr == nil && hErr == nil && weekday == "*" && isSingleDay(day):
		return fmt.Sprintf("Monthly on day %s at %02d:%02d", day, h, m)
	default:
		return cron
	}
}

func isSingleWeekday(w string) bool {
	n, err := strconv.Atoi(w)
	return err == nil && n >= 0 && n <= 6
}

func isSingleDay(d string) bool {
	n, err := strconv.Atoi(d)
	return err == nil && n >= 1 && n <= 31
}

func weekdayName(w string) string {
	names := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	n, _ := strconv.Atoi(w)
	return names[n]
}

func firewallFrom(proj *core.Project) Firewall {
	cfg := proj.Budget
	f := Firewall{
		SanitizerOn:      core.SanitizerEnabled(cfg),
		PIIMaskingOn:     core.PIIMaskingEnabled(cfg),
		CacheGuardOn:     core.CacheGuardEnabled(cfg),
		CircuitBreakerOn: core.CircuitBreakerEnabled(cfg),
	}
	if cfg != nil {
		f.MaxTokensPerRun = cfg.MaxTokensPerRun
		f.MaxMonthlyUSD = cfg.MaxMonthlyUSD
		if cfg.Sanitize != nil {
			f.DedupeLogsOn = cfg.Sanitize.DedupeLogs
			f.StripBlankOn = cfg.Sanitize.StripBlank
			f.StripCommentsOn = cfg.Sanitize.StripComments
		}
	}
	return f
}

// metricsFrom builds Metrics from a live orchestrator.Count result,
// matching each AgentMetrics to its AgentNode (by Name) so IsLocal
// carries over — see AgentMetrics's own doc comment for why that
// matters (conditional cost display, #12).
func metricsFrom(r orchestrator.CountResult, agents []AgentNode) *Metrics {
	isLocalByName := map[string]bool{}
	for _, a := range agents {
		isLocalByName[a.Name] = a.IsLocal
	}

	m := &Metrics{TotalTokens: r.TotalTokens, Costs: r.TotalCosts}
	for _, a := range r.Agents {
		am := AgentMetrics{Name: a.Agent, IsLocal: isLocalByName[a.Agent]}
		if a.Report != nil {
			fillAgentMetricsFromReport(&am, a.Report)
		}
		m.PerAgent = append(m.PerAgent, am)
	}
	if !r.IsGroup && r.Report != nil {
		var isLocal bool
		if len(agents) == 1 {
			isLocal = agents[0].IsLocal
		}
		am := AgentMetrics{Name: r.Name, IsLocal: isLocal}
		fillAgentMetricsFromReport(&am, r.Report)
		m.PerAgent = []AgentMetrics{am}
	}
	return m
}

// fillAgentMetricsFromReport copies every number the token-reduction
// pipeline (#18) needs straight from a real budget.Report — no
// recomputation, so it can never disagree with what `mova budget`
// itself would print for the same project right now.
func fillAgentMetricsFromReport(am *AgentMetrics, r *budget.Report) {
	am.Tokens = r.TotalTokens
	am.SanitizedLines = r.SanitizeStats.LinesRemoved
	am.PIIScanned = r.PIIStats.TokensScanned
	am.PIIMasked = r.PIIStats.TokensMasked
	am.CircuitTriggered = r.CircuitBreaker.RunExceeded || r.CircuitBreaker.MonthExceeded
	am.Costs = r.TotalCosts
	am.Components = r.Components
	am.DuplicatesRemoved = r.DuplicatesRemoved
	am.RawTokens = r.RawTokens
	am.RawCosts = r.RawCosts
	am.SavingsPercent = r.MemorySavingsPercent
	am.FocusComparison = r.Focus
}

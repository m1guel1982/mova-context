// trace.go — Run: the ONLY entry point CLI (cli/trace_cmd.go), MCP
// (mcp/trace_tool.go), HTTP (http/server.go), and Chat
// (cli/trace_cmd.go's runChatTrace) call to execute `context-trace` -
// this is what guarantees all four doors are identical: same
// analysis, same files written, same console text (see console.go).
package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/i18n"
)

// Options are the parameters of one context-trace run - the same
// shape no matter the door; each door is only responsible for parsing
// ITS OWN flags/arguments into this (see cli/trace_cmd.go,
// mcp/trace_tool.go).
type Options struct {
	Root    string // Mova Context install root (adapter/config/prices.json)
	Cwd     string // directory the command was invoked from (for a relative --output)
	Origin  string // "CLI" | "Chat" | "MCP" | "API HTTP"
	Project string // local project name (empty when using --repo)
	Task    string
	// IgnorePatterns: --ignore glob/extglob patterns (see
	// ignore_glob.go), applied at Discovery time before any
	// tokenization or PII/security evaluation - matched files count as
	// EXCLUDED with reason "excluded_by_ignore_pattern" and never enter
	// CANDIDATE, BM25 scoring, or cost estimation.
	IgnorePatterns []string
	// PruneDocstrings: --prune-docstrings — strip comments/docstrings
	// from each CANDIDATE file's content right before it's packaged
	// into the final context/token count, when the budget is tight
	// (see analyzer.go, astfilter.PruneDocs). Ranking (RankByTask)
	// still sees the FULL original content — pruning only ever
	// shrinks what gets counted/exported, never what gets scored.
	PruneDocstrings bool
	RepoURL        string // --repo <url> (empty for a local project)
	Branch         string // specific branch to clone; "" = the repository's default branch
	ExportFormat   string // "pdf" | "md" - defaults to DefaultExportFormat ("md")
	Output         string // --output; "" = each mode's own default (see runLocal/runRemote)
	// GenerateProjectJSON: when the analysis is remote and this is
	// true, writes the suggested project.json without asking for
	// interactive confirmation (CLI/Chat collect that confirmation
	// themselves BEFORE setting this to true - see
	// runContextTrace/runChatTrace).
	GenerateProjectJSON bool
	// OnProgress: optional status callback for a real terminal (CLI,
	// Chat) - see ProgressFunc's doc comment in types.go for why this
	// MUST stay nil for MCP/HTTP.
	OnProgress ProgressFunc

	// ── Audit Matrix (see Data.AgentClient/TargetModel/PolicyAuthor
	// in types.go) — each door (CLI, MCP, HTTP) fills what it knows;
	// whatever's left empty gets a safe default in applyAuditIdentity,
	// never left blank.
	AgentClient  string // "mova-cli" (CLI/Chat) or the real MCP client
	TargetModel  string // "<provider>/<model>" from the active llm_profile
	PolicyAuthor string // see core.ResolvePolicyAuthor
}

// Result is what Run returns - each door decides how to show it
// (console/chat prints Console as-is; MCP/HTTP pass it back verbatim
// as the response text).
type Result struct {
	Data                 *Data
	OutputDir            string
	Written              []string
	Console              string
	SuggestedProjectPath string // empty if nothing was generated (or it wasn't requested)
}

// Run executes the full analysis - local or remote depending on opts,
// writes the OUTPUT artifacts, and builds the console text - in that
// order, always cleaning up any temporary clone before returning (see
// the explicit cleanup call in runRemote).
// applyAuditIdentity fills in the 3 audit questions that depend on
// the entry door (who requested it?, which model received it?, who
// authorized the policy?) — see Data.AgentClient/TargetModel/
// PolicyAuthor. Each door (CLI, MCP, HTTP) fills what it knows in
// Options; this only applies the defaults so no field is ever left
// empty in the final report.
func applyAuditIdentity(d *Data, opts Options) {
	d.AgentClient = opts.AgentClient
	if d.AgentClient == "" {
		switch opts.Origin {
		case "MCP":
			d.AgentClient = "mcp-agent"
		default:
			d.AgentClient = "mova-cli"
		}
	}
	d.TargetModel = opts.TargetModel
	if d.TargetModel == "" {
		d.TargetModel = "n/a"
	}
	d.PolicyAuthor = opts.PolicyAuthor
	if d.PolicyAuthor == "" {
		d.PolicyAuthor = core.ResolvePolicyAuthor(opts.Root, opts.Project)
	}
}

func Run(adapter core.Adapter, opts Options) (*Result, error) {
	if opts.ExportFormat == "" {
		opts.ExportFormat = DefaultExportFormat
	}

	if opts.RepoURL != "" {
		return runRemote(opts)
	}
	return runLocal(adapter, opts)
}

func runLocal(adapter core.Adapter, opts Options) (*Result, error) {
	progress(opts, 10, "Reading project.json...")
	d, err := AnalyzeLocal(adapter, opts.Root, opts.Project, opts.Task, opts.Origin)
	if err != nil {
		return nil, err
	}
	applyAuditIdentity(d, opts)

	outDir := opts.Output
	if outDir == "" {
		// Local project default: the analyzed project's own root
		relOrAbsPath := filepath.FromSlash(d.ProjectJSONPath)
		if !filepath.IsAbs(relOrAbsPath) {
			relOrAbsPath = filepath.Join(opts.Root, relOrAbsPath)
		}
		outDir = filepath.Dir(relOrAbsPath)
	} else {
		outDir = NormalizeOutputPath(outDir, opts.Cwd)
	}

	progress(opts, 70, i18n.T("cli.messages.generating_reports"))
	written, err := WriteOutputs(d, opts.ExportFormat, outDir)
	if err != nil {
		return nil, err
	}
	progress(opts, 100, i18n.T("cli.messages.done"))

	return &Result{
		Data:      d,
		OutputDir: outDir,
		Written:   written,
		Console:   RenderConsole(d, OutputNames(written)),
	}, nil
}
func runRemote(opts Options) (*Result, error) {
	fetcher := NewFetcherFor(opts.RepoURL)
	progress(opts, 0, i18n.T("cli.messages.preparing_repo"))

	dir, branch, cleanup, err := fetcher.Fetch(opts.RepoURL, opts.Branch)
	if err != nil {
		return nil, err
	}
	cleanupDone := false
	defer func() {
		if !cleanupDone {
			cleanup()
		}
	}()

	progress(opts, 15, i18n.T("cli.messages.target_ready", map[string]any{"path": dir}))

	prices, err := budget.LoadPrices(opts.Root)
	if err != nil {
		return nil, err
	}

	d, err := AnalyzeRemote(dir, opts.RepoURL, branch, opts.Origin, opts.Root, opts.Task, opts.IgnorePatterns, prices, opts.PruneDocstrings, opts.OnProgress)
	if err != nil {
		return nil, err
	}
	applyAuditIdentity(d, opts)

	cleanup()
	cleanupDone = true

	// Solo muestra el mensaje de eliminación si la carpeta proviene de un clon en el directorio temporal
	if strings.HasPrefix(filepath.Clean(dir), filepath.Clean(os.TempDir())) {
		progress(opts, 85, fmt.Sprintf("Temporary directory removed: %s", dir))
	} else {
		progress(opts, 85, "Repository processing completed.")
	}

	outDir := opts.Output
	if outDir == "" {
		// Remote repository default: the current directory the CLI
		// was run from (see COMMANDS.md).
		outDir = NormalizeOutputPath("", opts.Cwd)
	} else {
		outDir = NormalizeOutputPath(outDir, opts.Cwd)
	}

	progress(opts, 90, i18n.T("cli.messages.generating_reports"))
	written, err := WriteOutputs(d, opts.ExportFormat, outDir)
	if err != nil {
		return nil, err
	}

	res := &Result{Data: d, OutputDir: outDir, Written: written}

	suggestedName := suggestProjectName(opts.RepoURL)
	suggested, err := BuildSuggestedProjectJSON(d, suggestedName)
	if err == nil {
		d.SuggestedProjectJSON = suggested
		d.SuggestedProjectName = suggestedName
	}

	if opts.GenerateProjectJSON && d.SuggestedProjectJSON != "" {
		path, err := WriteSuggestedProjectJSON(opts.Root, suggestedName, d.SuggestedProjectJSON)
		if err != nil {
			return nil, fmt.Errorf("could not generate the suggested project.json: %w", err)
		}
		res.SuggestedProjectPath = path
	}

	progress(opts, 100, i18n.T("cli.messages.done"))
	res.Console = RenderConsole(d, OutputNames(written))
	return res, nil
}

// progress calls opts.OnProgress when set - a tiny helper so every
// call site above reads as one line instead of a repeated nil check.
func progress(opts Options, percent int, message string) {
	if opts.OnProgress != nil {
		opts.OnProgress(percent, message)
	}
}

// suggestProjectName derives a reasonable project name from the
// repository URL ("https://github.com/fastapi/fastapi" -> "fastapi")
// - only to pre-fill the suggestion; the person can freely rename it
// in the generated project.json (and, before that, at the confirmation
// prompt - see cli/trace_cmd.go).
func suggestProjectName(url string) string {
	clean := filepath.Base(filepath.Clean(url))
	if clean == "." || clean == "/" || clean == "\\" || clean == "" {
		return "remote-project"
	}
	if len(clean) > len(".git") && clean[len(clean)-len(".git"):] == ".git" {
		clean = clean[:len(clean)-len(".git")]
	}
	return clean
}

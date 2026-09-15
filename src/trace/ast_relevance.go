// ast_relevance.go — the AST layer of context-trace's hybrid
// BM25 + AST relevance score (see relevance.go, which this file
// extends without changing its existing behavior for any file/
// language astfilter doesn't cover — see that fallback below). Fixes
// a real, reproducible failure mode: ranking "fix dependency
// injection" against fastapi put fastapi/security/oauth2.py above
// fastapi/dependencies/utils.py, purely because oauth2.py's
// docstrings repeat the word "dependency" more densely than
// utils.py's actual `solve_dependencies`/`get_dependant` code does —
// a lexical (BM25) blind spot no amount of structural-signal tuning
// in relevance.go fixes, because BM25 cannot tell a docstring from a
// declaration.
//
// This became possible once context-trace stopped needing CGO for
// AST work (see core/focus/astfilter, github.com/odvcencio/
// gotreesitter) — the honesty note at the top of relevance.go, about
// deliberately NOT doing real Tree-Sitter parsing, predates that
// migration and no longer applies to the languages astfilter covers.
// Every language astfilter/languages.go DOESN'T cover still gets
// pure BM25 + structural signals, exactly as before this file existed.
package trace

import (
	"path/filepath"
	"regexp"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"mova.local/core/focus/astfilter"
)

// astRoles a file can play — see docs/i18n/{es,en}/context-trace.md
// § "Producer vs Consumer" for the full explanation with the FastAPI
// example this file's header describes.
type astRole string

const (
	roleProducer astRole = "producer" // DECLARES the logic (function/class definitions matching the task's own verbs/nouns)
	roleConsumer astRole = "consumer" // only IMPORTS or ANNOTATES/CALLS something the task mentions, without declaring it
	roleUnknown  astRole = ""         // astfilter doesn't cover this language, or the file has neither signal
)

// astSignals is everything ast_relevance.go's scorer computed for one
// file — exported on RelevanceScore (see relevance.go) as real
// evidence, never a silent number.
type astSignals struct {
	DocRatio  float64 // fraction of the file's bytes inside comments/docstrings
	Role      astRole
	FuncNames []string // every function/method name declared, for the report's evidence line
}

// analyzeAST computes astSignals for one file, or ok=false when
// astfilter doesn't support this file's language — the caller then
// falls back to BM25 + structural signals alone, unchanged.
func analyzeAST(path, content string) (astSignals, bool) {
	if !astfilter.Supported(path) {
		return astSignals{}, false
	}
	tree, lang, ok := astfilter.ParseTree([]byte(content), path)
	if !ok {
		return astSignals{}, false
	}
	docBytes := docCommentBytes(tree, lang, []byte(content))
	total := len(content)
	ratio := 0.0
	if total > 0 {
		ratio = float64(docBytes) / float64(total)
	}
	names := astfilter.AllNames(tree, lang, []byte(content), path, "func")
	return astSignals{DocRatio: ratio, FuncNames: names}, true
}

// docCommentBytes sums the byte length of every `comment` node plus
// every "docstring-shaped" statement — a `string` literal that is the
// FIRST statement of a block/module, exactly the user's own spec
// ("expression_statement que contenga string"): a bare string used as
// a real expression (assigned, passed as an argument, concatenated)
// is never in that position, only a genuine docstring is.
func docCommentBytes(tree *gts.Tree, lang *gts.Language, content []byte) int {
	total := 0
	var walk func(n *gts.Node)
	walk = func(n *gts.Node) {
		if n == nil {
			return
		}
		t := n.Type(lang)
		if t == "comment" {
			total += int(n.EndByte() - n.StartByte())
		}
		if (t == "block" || t == "module" || t == "program") && n.ChildCount() > 0 {
			first := n.Child(0)
			if first != nil && first.Type(lang) == "string" {
				total += int(first.EndByte() - first.StartByte())
			}
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	return total
}

// producerVerbs are the built-in "this function DEFINES core logic"
// vocabulary the role classifier checks a file's OWN declared
// function names against, in addition to the task's own words (a
// task mentioning "solve_dependencies" already boosts via
// astScoreAndRole's name-match step below — this list is what lets a
// file be recognized as a producer even for a task that never spells
// out the exact function name, e.g. plain "fix dependency injection").
var producerVerbs = regexp.MustCompile(`(?i)^(resolve|solve|build|compute|generate|construct|create|init|make|get_dependant|get_dependency|dependant)`)

// consumerCallPattern recognizes an annotation/DI-style call at
// argument position — Depends(...), Inject(...), @inject, Provide(...)
// — the "just USES it" signal a Consumer file has instead of a
// declaration. This is intentionally a small, extensible list, not an
// attempt to cover every DI framework.
var consumerCallPattern = regexp.MustCompile(`\b(Depends|Inject|Provide|Autowired)\s*\(`)

// astScoreAndRole computes the boost (or penalty) astSignals earns
// against a task's own tokens, and classifies the file's role. boost
// is ADDED to BM25+structural (see relevance.go's RankByTask); the
// DocRatio penalty is applied as a MULTIPLICATIVE factor on the
// combined total, per the request's own "aplica un factor de
// penalización" wording — a 60%-documentation file has its total
// score roughly halved, never zeroed outright (a file that is mostly
// comments ABOUT the task is still weak evidence, not zero evidence).
func astScoreAndRole(sig astSignals, content string, taskTerms map[string]bool) (boost float64, role astRole, penalty float64) {
	penalty = 1.0
	if sig.DocRatio > 0.30 {
		penalty = 1.0 - sig.DocRatio*0.8 // e.g. 60% docs -> ×0.52, 90% docs -> ×0.28
		if penalty < 0.15 {
			penalty = 0.15 // never fully zero out — see doc comment above
		}
	}

	producerHit, taskNameHit := false, false
	for _, name := range sig.FuncNames {
		lower := strings.ToLower(name)
		if taskTerms[lower] {
			boost += 4.0 // the task named this exact function — strongest possible AST signal
			taskNameHit = true
		}
		if producerVerbs.MatchString(name) {
			producerHit = true
		}
	}
	if producerHit || taskNameHit {
		role = roleProducer
		boost += 3.0
	} else if consumerCallPattern.MatchString(content) {
		role = roleConsumer
		boost -= 1.5
	}
	return boost, role, penalty
}

// parseTaskHints extracts the extended --task syntax this feature
// adds on top of plain free-text tasks (both keep working — a task
// with none of these markers behaves exactly as before):
//
//	"solve_dependencies get_dependant in fastapi/dependencies/utils.py"
//	"resolve engine focus:AST_producer"
//
// explicitPath is the file named after " in ", if any — that file
// gets an overwhelming boost (see RankByTask) so it reliably lands at
// the top when the person already knows exactly where to look.
// producerOnly is true when a "focus:ast_producer" (case-insensitive)
// marker is present — candidates classified roleConsumer are then
// zeroed out rather than merely penalized. bareTask is the task text
// with both markers stripped, for ordinary BM25 tokenization.
func parseTaskHints(task string) (bareTask, explicitPath string, producerOnly bool) {
	bareTask = task
	if m := regexp.MustCompile(`(?i)\bfocus:ast_producer\b`).FindString(bareTask); m != "" {
		producerOnly = true
		bareTask = strings.TrimSpace(strings.Replace(bareTask, m, "", 1))
	}
	if m := regexp.MustCompile(`(?i)\bin\s+([^\s]+\.[A-Za-z0-9]+)\b`).FindStringSubmatch(bareTask); m != nil {
		explicitPath = m[1]
		bareTask = strings.TrimSpace(strings.Replace(bareTask, m[0], "", 1))
	}
	return bareTask, explicitPath, producerOnly
}

// explicitPathBoost is the score given to a candidate whose path
// matches (exactly, or by suffix — "utils.py" also matches
// "fastapi/dependencies/utils.py") the file named in a
// "... in <path>" task, chosen to be larger than any realistic
// BM25+AST combination so that file reliably ranks #1 (see this
// feature's own "Fase AST" description: "pasará automáticamente al
// Top 1").
const explicitPathBoost = 1000.0

func matchesExplicitPath(candidate, explicitPath string) bool {
	if explicitPath == "" {
		return false
	}
	c := filepath.ToSlash(candidate)
	e := filepath.ToSlash(explicitPath)
	return c == e || strings.HasSuffix(c, "/"+e) || filepath.Base(c) == e
}

// relevance.go — the task-relevance narrowing step context-trace's
// own docs already flagged as a documented gap ("CANDIDATE currently
// equals every DISCOVERED file"). When --task/-t is given, this
// scores every file against the task text with BM25 (a well-known,
// deterministic lexical ranking formula - no embeddings, no network
// call, no model) plus a regex structural signal (imports/declaration
// lines) AND, for languages astfilter covers, a REAL AST layer (see
// ast_relevance.go: docstring-density penalty, function-name boost,
// producer/consumer role) - only files clearing a computed cutoff
// stay CANDIDATE; the rest become EXCLUDED with reason "Not relevant
// to task" - a REAL reduction, not merely the security-driven one
// this package already reported.
//
// Update (see ast_relevance.go's own header): the "not a real
// Tree-Sitter parser" limitation this comment used to document no
// longer applies for any language core/focus/astfilter covers - that
// became possible once context-trace stopped needing CGO for AST work
// (github.com/odvcencio/gotreesitter is pure Go). For any OTHER
// language, the fallback below (structuralSignals: line-level
// import/declaration pattern matching) is still exactly what runs -
// a real, useful signal, just not a syntax tree.
package trace

import (
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// RelevanceScore is one file's BM25 + structural-signal score against
// a task query - exported so the report/console can show it as
// evidence, never a silent black-box cutoff.
type RelevanceScore struct {
	Path            string
	BM25            float64
	StructuralBoost float64
	ASTBoost        float64 // see ast_relevance.go — 0 when astfilter doesn't cover this file's language
	ASTDocRatio     float64 // fraction of the file that is comments/docstrings (evidence for the penalty applied to Total)
	ASTRole         string  // "producer" | "consumer" | "" (unclassified or unsupported language)
	Total           float64
}

// tokenize is a tiny, deterministic word splitter shared by both the
// corpus (file contents) and the query (task text): lowercase,
// alphanumeric runs only. It is intentionally NOT a real code
// tokenizer (no identifier-casing awareness) - simplicity here keeps
// the whole scorer auditable in a few lines, matching the same
// "structural, not semantic" philosophy as sanitize/secrets.go.
var tokenRe = regexp.MustCompile(`[A-Za-z0-9_]+`)

func tokenize(s string) []string {
	matches := tokenRe.FindAllString(strings.ToLower(s), -1)
	return matches
}

// bm25Corpus precomputes what BM25 needs once per run: per-document
// term frequencies, document lengths, and document frequency per
// term - standard Robertson/Sparck-Jones BM25 (k1=1.5, b=0.75, the
// widely used defaults; see Robertson & Zaragoza, "The Probabilistic
// Relevance Framework: BM25 and Beyond", 2009).
type bm25Corpus struct {
	docTermFreq []map[string]int
	docLen      []int
	avgDocLen   float64
	docFreq     map[string]int
	n           int
}

const bm25K1 = 1.5
const bm25B = 0.75

func newBM25Corpus(docs []string) *bm25Corpus {
	c := &bm25Corpus{docFreq: map[string]int{}, n: len(docs)}
	totalLen := 0
	for _, doc := range docs {
		terms := tokenize(doc)
		tf := map[string]int{}
		for _, t := range terms {
			tf[t]++
		}
		c.docTermFreq = append(c.docTermFreq, tf)
		c.docLen = append(c.docLen, len(terms))
		totalLen += len(terms)
		for t := range tf {
			c.docFreq[t]++
		}
	}
	if c.n > 0 {
		c.avgDocLen = float64(totalLen) / float64(c.n)
	}
	return c
}

// score returns the BM25 score of document i against queryTerms.
func (c *bm25Corpus) score(i int, queryTerms []string) float64 {
	if c.avgDocLen == 0 {
		return 0
	}
	var total float64
	dl := float64(c.docLen[i])
	for _, term := range queryTerms {
		df := c.docFreq[term]
		if df == 0 {
			continue
		}
		idf := math.Log(1 + (float64(c.n)-float64(df)+0.5)/(float64(df)+0.5))
		tf := float64(c.docTermFreq[i][term])
		denom := tf + bm25K1*(1-bm25B+bm25B*dl/c.avgDocLen)
		if denom == 0 {
			continue
		}
		total += idf * (tf * (bm25K1 + 1) / denom)
	}
	return total
}

// structuralSignals extracts import/require and function/class/def
// declaration lines - see this file's header for why this, and not a
// real parser. Language family is inferred from the file extension.
var importPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\s*(?:import|from|require|use|#include|package)\s+["'\w./\\-]+`),
}
var declPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\s*(?:func|def|class|function|interface|type|struct)\s+([A-Za-z_][A-Za-z0-9_]*)`),
}

func structuralSignals(content string) []string {
	var symbols []string
	for _, re := range append(append([]*regexp.Regexp{}, importPatterns...), declPatterns...) {
		for _, m := range re.FindAllStringSubmatch(content, -1) {
			line := m[0]
			for _, t := range tokenize(line) {
				if len(t) > 2 {
					symbols = append(symbols, t)
				}
			}
		}
	}
	return symbols
}

// RankByTask scores every (path, content) pair against task and
// returns a RelevanceScore per file, plus the cutoff score (for
// display/debugging only - see SelectBelowCutoff for the actual,
// tie-safe exclusion decision).
func RankByTask(files map[string]string, task string, cutoffPercentile float64) (map[string]RelevanceScore, float64) {
	if task == "" || len(files) == 0 {
		return nil, 0
	}
	// Extended --task syntax (see ast_relevance.go's parseTaskHints):
	// "<symbols> in <path>" and a trailing "focus:ast_producer"
	// marker. bareTask (markers stripped) is what BM25 tokenizes —
	// exactly the old behavior for a task that uses neither marker.
	bareTask, explicitPath, producerOnly := parseTaskHints(task)
	paths := make([]string, 0, len(files))
	docs := make([]string, 0, len(files))
	for p, c := range files {
		paths = append(paths, p)
		docs = append(docs, c)
	}
	corpus := newBM25Corpus(docs)
	queryTerms := tokenize(bareTask)
	taskTermSet := map[string]bool{}
	for _, t := range queryTerms {
		taskTermSet[t] = true
	}

	scores := make(map[string]RelevanceScore, len(paths))
	values := make([]float64, 0, len(paths))
	for i, p := range paths {
		bm25 := corpus.score(i, queryTerms)
		boost := 0.0
		for _, sym := range structuralSignals(docs[i]) {
			if taskTermSet[sym] {
				boost += 1.0
			}
		}
		// Filename itself matching the task is a strong, cheap signal
		// (e.g. task "fix login bug" vs a file named auth/login.go).
		base := strings.ToLower(filepath.Base(p))
		for t := range taskTermSet {
			if len(t) > 2 && strings.Contains(base, t) {
				boost += 2.0
			}
		}

		astBoost, docRatio, role := 0.0, 0.0, astRole("")
		penalty := 1.0
		if sig, ok := analyzeAST(p, docs[i]); ok {
			astBoost, role, penalty = astScoreAndRole(sig, docs[i], taskTermSet)
			docRatio = sig.DocRatio
		}
		total := (bm25 + boost + astBoost) * penalty
		if producerOnly && role == roleConsumer {
			total = 0 // "focus:ast_producer" — Consumers are dropped, not merely penalized
		}
		if matchesExplicitPath(p, explicitPath) {
			total += explicitPathBoost
		}
		scores[p] = RelevanceScore{
			Path: p, BM25: bm25, StructuralBoost: boost,
			ASTBoost: astBoost, ASTDocRatio: docRatio, ASTRole: string(role),
			Total: total,
		}
		values = append(values, total)
	}

	return scores, percentileCutoff(values, cutoffPercentile)
}

// SelectBelowCutoff picks which files should be treated as "not
// relevant to task" — the BOTTOM floor(n*cutoffPercentile) files by
// score, selected by RANK rather than by comparing against a single
// threshold value. A pure value-threshold comparison silently selects
// nothing whenever many files tie at the same (often zero) score — a
// real, verified failure mode with small/uniform corpora — so ties at
// the boundary are broken deterministically by path name, and exactly
// the intended count is always excluded (never zero just because of a
// tie, and never more than requested).
func SelectBelowCutoff(scores map[string]RelevanceScore, cutoffPercentile float64) map[string]bool {
	excluded := map[string]bool{}
	n := len(scores)
	if n == 0 || cutoffPercentile <= 0 {
		return excluded
	}
	paths := make([]string, 0, n)
	for p := range scores {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool {
		si, sj := scores[paths[i]].Total, scores[paths[j]].Total
		if si != sj {
			return si < sj
		}
		return paths[i] < paths[j] // deterministic tie-break
	})
	cut := int(float64(n) * cutoffPercentile)
	if cut > n {
		cut = n
	}
	// Additionally, ALWAYS exclude every file with a score of exactly
	// zero (no lexical/structural overlap with the task whatsoever) -
	// regardless of the percentile cutoff. Without this, a fixed
	// percentile keeps excluding the same PROPORTION of files no
	// matter how irrelevant the task is to the whole repository (a
	// verified failure mode: an unrelated task like "implement a
	// PostgreSQL payment system" against a web-framework repo left
	// 65% of files as CANDIDATE, most scoring zero, because the
	// percentile alone never looks at the actual score values). Since
	// paths are sorted ascending by score and no score is negative,
	// every zero-score entry is contiguous at the start of the slice.
	zeroBoundary := 0
	for zeroBoundary < n && scores[paths[zeroBoundary]].Total <= 0 {
		zeroBoundary++
	}
	if zeroBoundary > cut {
		cut = zeroBoundary
	}
	for i := 0; i < cut; i++ {
		excluded[paths[i]] = true
	}
	return excluded
}

// topRelevanceRows builds the top-N RelevanceRankRow list for the
// report's ranking section, sorted by score descending — returns nil
// when scores is nil (no --task given), so callers can render nothing
// unconditionally. Token counts here are an approximate word-count
// (not the real BPE count) purely for this display row; the
// authoritative token accounting is StateTotals, computed once in the
// main analysis loop.

// fraction of files fall - simple sort-based percentile, no
// approximation needed at the file counts context-trace deals with.
func percentileCutoff(values []float64, cutoffPercentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1] > sorted[j]; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	idx := int(float64(len(sorted)) * cutoffPercentile)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

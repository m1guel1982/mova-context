// closure.go — types and hook for dependency-closure validation.
// The implementation lives in mova.local/graph (it reuses the dependency
// graph's parser and edge resolution) and registers itself in
// ClosureHook at init, the same pattern as GraphHook: core and budget
// cannot import graph (graph → diagram → budget → core).
package core

// Dependency is one edge from a focused symbol to a symbol NOT in the
// context.
type Dependency struct {
	From     string `json:"from"`   // "file::symbol" in focus
	To       string `json:"to"`     // "file::symbol" depended upon
	Kind     string `json:"kind"`   // "call" | "ref"
	Status   string `json:"status"` // "excluded" | "outside"
	Inferred bool   `json:"inferred,omitempty"`
	Accepted bool   `json:"accepted,omitempty"`
	Reason   string `json:"reason,omitempty"` // accept_missing reason
}

// ClosureReport is the dependency-closure result for one task.
type ClosureReport struct {
	Task      string       `json:"task"`
	Policy    string       `json:"policy"` // block | warn | off
	Checked   bool         `json:"checked"`
	Focused   int          `json:"focused_symbols"`
	Conflicts []Dependency `json:"conflicts"` // depend on EXCLUDED symbols (not accepted)
	Accepted  []Dependency `json:"accepted,omitempty"`
	Outside   []Dependency `json:"outside"` // depend on repo symbols not in the context
	Note      string       `json:"note,omitempty"`
}

// Blocking reports whether this report must stop the release.
func (r ClosureReport) Blocking() bool {
	return r.Policy == "block" && len(r.Conflicts) > 0
}

// ClosureHook is set by mova.local/graph. nil (graph not linked) → no
// validation, reported as such by the callers.
var ClosureHook func(root string, proj *Project, taskName string) []ClosureReport

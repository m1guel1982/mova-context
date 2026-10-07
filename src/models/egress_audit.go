// egress_audit.go — the ONE implementation of "egress_audit" (see
// project.json's optional "egress_audit" block, core.EgressAuditConfig,
// core.ResolveEgressAudit), shared verbatim by every door that can hand
// finished context to a network call or an MCP tool result: CLI
// `mova chat` and `mova run` (cli/chat_cmd.go), MCP's chat_completion
// AND get_full_context tools (mcp/chat_tool.go, mcp/context_tool.go),
// and HTTP (a thin wrapper over the same MCP tools — see
// http/server.go). There is only ever ONE condition that activates
// this: "dry_run": true. There is no separate "enabled" switch —
// dry_run alone is both necessary and sufficient, on purpose, so there
// is exactly one flag to reason about.
//
// Pipeline position (per PROJECT_JSON.md § egress_audit and this
// package's README-linked diagram):
//
//	selected context → existing governance → existing sanitization →
//	egress_audit (THIS FILE) → LLM provider / MCP tool result
//
// By the time EgressGate or Session.applyEgressAudit run, the context
// they see is already governed+sanitized (built once by
// budget.BuildGatedContext or core.BuildContextSections and handed in
// as plain text) — this file never re-sanitizes or re-governs
// anything. It only decides: log it, and/or block it from leaving.
package models

import (
	"mova.local/budget"
	"mova.local/i18n"
)

// buildAirgapMessage renders the full air-gap block returned to
// whoever asked for context while dry_run is active: the audit header
// ("reports.egress_airgap_message" — dry_run/tokens_evaluated/
// tokens_sent) plus the imperative anti-bypass directive
// ("reports.egress_airgap_directive"), added per PROBLEM 2 in the QA
// fix history (Test 1: an MCP host that received only the header used
// to go read context-report.md/project.json itself to "reconstruct"
// the context instead of honoring the block — see docs/README.md's
// note on host-side limitations for why Mova can describe but never
// fully GUARANTEE that behavior).
//
// The directive key is looked up SEPARATELY and appended only if it
// resolves to real text — i18n.T's documented fallback (active lang ->
// "en" -> the literal key string) means an operator can:
//   - customize the directive's wording per language, hot (edit
//     config/lang/{es,en}.json — see i18n_reload.go's content-hot-
//     reload) — this is what "totalmente personalizable... en
//     caliente" means here: it takes effect on the *next* dry_run
//     block within ~1s of saving, with no restart;
//   - or delete/blank the key entirely to turn the directive off
//     without breaking the base audit block, since a value of ""
//     and a wholly-missing key both make this function skip it.
//
// This is why the check is `directive != "" && directive != the key
// itself` rather than assuming the key exists.
func buildAirgapMessage(tokens int) string {
	msg := i18n.T("reports.egress_airgap_message", map[string]any{"tokens": tokens})
	directive := i18n.T("reports.egress_airgap_directive")
	if directive != "" && directive != "reports.egress_airgap_directive" {
		msg = msg + "\n\n" + directive
	}
	return msg
}

// EgressGateResult is what every door needs after checking
// egress_audit for one piece of finished context.
type EgressGateResult struct {
	// Blocked is true when dry_run is active — the caller MUST NOT
	// include contextText (or any derivative of it) in whatever it
	// returns; it returns Message instead. This is the one field every
	// caller (Session.Send, chat_completion, get_full_context) branches
	// on — see this file's package comment for why dry_run alone
	// decides it.
	Blocked bool
	// Message is the static, translated, information-only text to
	// return INSTEAD of contextText when Blocked — see buildAirgapMessage
	// and config/lang/{es,en}.json's "reports.egress_airgap_message" /
	// "reports.egress_airgap_directive".
	Message string
	// Tokens is how many tokens contextText measured at — included in
	// Message and exposed here too for callers that also want it in
	// their own console/status output.
	Tokens int
}

// EgressGate is the ONE check every door performs before letting
// finished context leave Mova. dryRun/outputFile come from
// core.ResolveEgressAudit — this function doesn't re-resolve
// project.json itself, so it has no need to import core.Project and
// stays reusable from any package that already has those two values
// (mcp/chat_tool.go, mcp/context_tool.go, cli/chat_cmd.go). modelHint
// (may be "") is only used to pick a tokenizer encoding — see
// budget.CountTokens — never to decide whether to call a model.
//
// Returns a non-nil error ONLY when a configured output_file could not
// be written — and in that case the caller must treat it exactly like
// any other fatal governance failure: stop, do not return contextText,
// do not return a Blocked=false result. A write failure is not a
// softer case than dry_run; it is the same "evidence must exist before
// anything leaves" rule the whole feature is for.
func EgressGate(dryRun bool, contextText, modelHint string) (EgressGateResult, error) {
	tokens, _, _ := budget.CountTokens(contextText, modelHint)
	if !dryRun {
		return EgressGateResult{Blocked: false, Tokens: tokens}, nil
	}
	return EgressGateResult{
		Blocked: true,
		Tokens:  tokens,
		Message: buildAirgapMessage(tokens),
	}, nil
}

// egressAuditOutcome tells Send/SendStream what to do right after
// egress_audit ran: stop (dry-run or a write failure) or proceed to
// the provider call as usual.
type egressAuditOutcome struct {
	stop   bool
	reply  string
	err    error
	dryRun bool
}

// applyEgressAudit is called once per Send/SendStream, right before
// the provider would be invoked. It never mutates s.System/s.History —
// callers are responsible for any rollback (see Send/SendStream, which
// already roll back the just-appended user turn on any non-success
// path, exactly like they do today for a provider error). This is
// Session's own thin wrapper around the same rule EgressGate applies —
// kept separate because Session already has its dry_run/output_file
// pre-resolved once at chat setup (see core.ResolveEgressAudit's
// call sites in cli/chat_cmd.go, cli/chat_helpers.go, mcp/chat_tool.go)
// rather than re-resolved from project.json on every single message.
func (s *Session) applyEgressAudit() egressAuditOutcome {
	tokens, _, _ := budget.CountTokens(s.System, s.Model)
	if s.EgressAuditDryRun {
		_ = s.Run.Event("dry_run_block", map[string]any{"tokens_estimated": tokens, "history_messages": len(s.History)})
		msg := buildAirgapMessage(tokens)
		return egressAuditOutcome{stop: true, reply: msg, dryRun: true}
	}
	return egressAuditOutcome{}
}

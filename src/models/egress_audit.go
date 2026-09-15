// egress_audit.go — the ONE implementation of "egress_audit" (see
// project.json's optional "egress_audit" block, core.EgressAuditConfig,
// core.ResolveEgressAudit) shared by every door that ends up calling
// Session.Send/SendStream: CLI `mova chat`, MCP's chat_completion tool,
// and HTTP (which is a thin wrapper over the same MCP tool) — see
// applyEgressAudit's call sites in chat.go.
//
// Pipeline position (per PROJECT_JSON.md § egress_audit):
//
//	selected context → existing governance → existing sanitization →
//	egress_audit → LLM provider
//
// Session.System already IS "the context selected → governed →
// sanitized" by the time Send/SendStream run (it's built once by
// budget.BuildGatedContext and handed to SetSystem — see
// cli/chat_cmd.go, cli/chat_helpers.go, mcp/chat_tool.go), so this file
// never re-sanitizes or re-governs anything: it only logs that final
// text and/or stops before the provider call.
package models

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mova.local/trace"
)

// DryRunReply is returned as the "reply" by Send/SendStream when
// egress_audit's dry_run is true — a successful, real response telling
// the caller (CLI/MCP/HTTP, all read this the same way a normal reply
// works) that sanitization/auditing completed and no inference ran.
const DryRunReply = "[egress_audit] dry-run completed: context was governed, sanitized" +
	" and (if configured) logged. No LLM provider was called."

// egressAuditOutcome tells Send/SendStream what to do right after
// egress_audit ran: stop (dry-run or a write failure) or proceed to
// the provider call as usual.
type egressAuditOutcome struct {
	stop  bool
	reply string
	err   error
}

// applyEgressAudit is called once per Send/SendStream, right before
// the provider would be invoked. It never mutates s.System/s.History —
// callers are responsible for any rollback (see Send/SendStream, which
// already roll back the just-appended user turn on any non-success
// path, exactly like they do today for a provider error).
func (s *Session) applyEgressAudit() egressAuditOutcome {
	if s.EgressAuditOutputFile != "" {
		if err := writeEgressAuditLog(s.EgressAuditOutputFile, s.System); err != nil {
			return egressAuditOutcome{stop: true, err: fmt.Errorf("egress_audit: could not write %s: %w", s.EgressAuditOutputFile, err)}
		}
	}
	if s.EgressAuditDryRun {
		return egressAuditOutcome{stop: true, reply: DryRunReply}
	}
	return egressAuditOutcome{}
}

// writeEgressAuditLog appends ONE clearly delimited execution block —
// execution_id, UTC timestamp, and the sanitized context verbatim — to
// outputFile, creating any missing directories first. Always APPENDS
// (os.O_APPEND|os.O_CREATE), so a previous run's log is never
// truncated or overwritten; each block is fenced with its own
// execution_id so concatenated runs stay unambiguous to a human or a
// script reading the file. sanitizedContext is written exactly as
// received — this function never sees, and cannot accidentally write,
// a pre-sanitization/raw version, since callers only ever pass
// Session.System (see applyEgressAudit above).
func writeEgressAuditLog(outputFile, sanitizedContext string) error {
	dir := filepath.Dir(outputFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", outputFile, err)
	}
	defer f.Close()

	executionID := trace.NewExecutionID()
	timestamp := time.Now().UTC().Format(time.RFC3339)

	block := fmt.Sprintf(
		"\n## Egress audit — execution %s\n\n- execution_id: `%s`\n- timestamp: %s (UTC)\n\n"+
			"### Sanitized context sent to the LLM\n\n```text\n%s\n```\n",
		executionID, executionID, timestamp, sanitizedContext,
	)

	if _, err := f.WriteString(block); err != nil {
		return fmt.Errorf("writing %s: %w", outputFile, err)
	}
	return nil
}

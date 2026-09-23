// context_tool.go — MCP/HTTP tool "get_full_context" (= `mova run`).
//
// Before returning anything, the FULL Context Governance pipeline runs
// — Sanitizer → PII Masking → Circuit Breaker → the existing
// "budget": {"max_tokens": N} gate — via budget.BuildGatedContext, the
// exact same chokepoint cli/run_cmd.go ("mova run") and
// chat_tool.go's chat_completion use. This tool used to call
// core.BuildContextSections directly instead, which builds the RAW
// sections (no PII Masking, no Sanitizer/Circuit Breaker bookkeeping)
// — see PROBLEM 1 in the QA fix history below for why that was a bug,
// not just a style inconsistency.
//
// Air-gap: this is also a context-EXPOSING tool, so it goes through
// models.EgressGate exactly like chat_completion does — "egress_audit":
// {"dry_run": true} blocks the context from ever reaching this tool's
// result, the same as it blocks chat_completion. There is no separate
// "enabled" switch: dry_run alone activates it, everywhere. Crucially,
// models.EgressGate (via WriteEgressAuditLog) ALSO writes the on-disk
// evidence file (egress_audit.output_file) whenever one is configured,
// independent of dry_run — so the text passed into EgressGate here
// must already be the governed/masked text, or that on-disk file leaks
// PII even when dry_run:true correctly blocks the network egress.
//
// PROBLEM 1 fix (PII leak in on-disk evidence, QA Test 6): this
// function previously did:
//
//	sections, _ := core.BuildContextSections(adapter, root, project, task)
//	ctxText := sections.Full()
//	... EgressGate(dryRun, outputFile, ctxText, modelHint)
//
// core.BuildContextSections never runs budget.applyPIIMasking (that
// stage only exists inside budget.BuildGatedContext, see
// budget/gated_context.go). So even with
// project.json's "budget.pii_masking.enabled": true, ctxText carried
// raw customer PII straight from customers.json/customer-profile.pdf
// — both back to the MCP caller (when dry_run:false) and into
// WriteEgressAuditLog's on-disk block (egress_audit.output_file,
// regardless of dry_run). Confirmed as H2 ("a parallel pipeline that
// never passes through sanitization"), not H1 — MaskPII itself does
// substitute the buffer correctly (see sanitize/pii.go MaskPII and
// budget/gated_context.go's applyPIIMasking, which mutates
// sections.Focus/Memory in place BEFORE sections.Full() is read); the
// bug was that this one door skipped calling that stage altogether.
// chat_tool.go's chatCompletionTool already called
// budget.BuildGatedContext correctly, which is why chat_completion's
// own audit-log blocks were properly masked while get_full_context's
// were not. Fixed by routing through budget.BuildGatedContext like
// every other door, so PII Masking (and Sanitizer/Circuit Breaker) run
// before ANY text reaches the tool result or the disk.
package mcp

import (
	"mova.local/budget"
	"mova.local/core"
	"mova.local/models"
)

// fullContextTool builds the project's context through the full
// Context Governance pipeline (budget.BuildGatedContext — Sanitizer →
// PII Masking → Circuit Breaker → Budget gate), then applies the
// egress air-gap gate, before returning it.
func fullContextTool(adapter core.Adapter, root, project, task string) (string, error) {
	proj, err := adapter.GetProject(project)
	if err != nil {
		return "", err
	}

	// budget.BuildGatedContext runs the full Context Governance
	// pipeline — the exact same pipeline `mova run`/chat_completion
	// already go through, so get_full_context never has its own copy
	// of "build then gate" that can silently skip a stage (see the
	// PROBLEM 1 fix note above). Err covers Sanitizer/PII/Circuit
	// Breaker failures AND the Budget "max_tokens" gate — nothing
	// ever reaches the caller (Claude Console, Codex, Gemini, a
	// script...) without every one of those checks first.
	gated := budget.BuildGatedContext(adapter, root, project, task)
	if gated.Err != nil {
		return "", gated.Err
	}

	modelHint := ""
	if proj.LLMProfile != nil {
		modelHint = proj.LLMProfile.Config
	}

	// Air-gap gate — see the package comment above. Runs AFTER every
	// Context Governance stage, so the text it ever sees — whether it
	// goes out to the caller or into WriteEgressAuditLog's on-disk
	// evidence — is always the fully governed/masked gated.Text, never
	// core.BuildContextSections' raw sections.Full() output.
	dryRun, outputFile := core.ResolveEgressAudit(root, project, proj)
	gateResult, gerr := models.EgressGate(dryRun, outputFile, gated.Text, modelHint)
	if gerr != nil {
		return "", gerr
	}
	if gateResult.Blocked {
		return gateResult.Message, nil
	}

	return gated.Text, nil
}

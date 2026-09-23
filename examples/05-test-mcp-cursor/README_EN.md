Dry-run Evidence (Test in Cline — Anthropic Haiku)

This test verifies Mova's context governance control using egress_audit.dry_run: true on the /projects/02-pii-compliance-governance project.

Mova processed the source files (customers.json, customer-profile.pdf, etc.), applied PII masking, and calculated the context size (7,787 tokens). Since dry-run mode was active, it generated the audit evidence report (egress.md) on disk without sending any tokens to the external LLM.

Claude Anthropic Haiku (via Cline) acted solely as the client interface: it received the security directive from Mova's MCP server and displayed the diagnosis on screen without attempting to read local files or bypass the block. The engine that measured, sanitized, and enforced the control was Mova.


"egress_audit": {
  "dry_run": true,
  "output_file": "egress_sanitized.md"
}
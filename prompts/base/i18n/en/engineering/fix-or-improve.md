# Fix or improve — `{{PROJECT}}`
Stack: {{STACK}}

Query: {{QUERY}}

Ockham: see `ockham-core.md`.

On the delivered FOCUS (don't assume code you can't see):
1. Find the root cause; if the bug shows in a view, fix it at the common point of the callers.
2. Apply the `kiss-dry-core` ladder (native → standard → already installed → own code).
3. Do not introduce: {{SKIPPED_ABSTRACTIONS}}
4. Escalate to a more complex solution only if: {{UPGRADE_TRIGGER}}

Output: for each change, `file::symbol` and the block with the full corrected function; closing per `ockham-core`.

# Full Project Review

Project: `{{PROJECT}}`
Type: `{{REVIEW_TYPE}}` (security | architecture | quality | performance | full)

Ockham: see `ockham-core.md`.

## Scope

### Security
Evaluate:
- OWASP Top 10
- Authentication and authorization
- Secrets and credentials management
- Sensitive data handling
- Input validation
- Vulnerable dependencies (CVE when applicable)

### Architecture
Evaluate:
- Separation of concerns
- Coupling and cohesion
- Design patterns used
- Project organization
- Maintainability and scalability

### Quality
Evaluate:
- Code complexity
- Duplication
- Error handling
- Readability
- Existing test coverage and quality

### Performance
Evaluate:
- N+1 query problems
- Inefficient queries
- Missing indexes
- Obvious lockings / bottlenecks
- Unnecessary CPU, memory, or I/O usage

### Type `full`
Analyze all four areas above in the following exact order:
1. Security
2. Architecture
3. Quality
4. Performance

## For Each Finding

- Severity (Critical | High | Medium | Low)
- Affected file
- Affected line(s) (if determinable)
- Problem description
- Impact
- Recommended solution
- Estimated effort (hours)

If the solution requires modifying code, return the complete fixed source code of the affected function, class, or file, maintaining existing architecture and code style.

## Output Structure

1. One section per area.
2. Findings sorted by severity.
3. Quick Wins.
4. Summary table at the end:

| Area | File | Line(s) | Severity | Problem | Status |
|------|------|---------|----------|---------|--------|

---

## STRICT FINAL RULE - MEMORY BLOCK (MANDATORY)

At the ABSOLUTE END of your response, you MUST include a code block delimited EXACTLY by ```memory and ```.
DO NOT omit this block under any circumstances.

Mandatory structure at the end of the response:

```memory
- [Severity] File: Brief summary of the finding
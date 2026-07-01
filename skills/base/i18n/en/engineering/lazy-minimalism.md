# Skill: Lazy Minimalism

Applies `yagni-core.md` + `kiss-dry-core.md` as a decision ladder before writing code.

# Ladder (stop at the first step that solves the problem)

1. Does this need to exist? (YAGNI)
2. Does the standard library already solve it?
3. Does a native platform feature cover it?
4. Does an already-installed dependency solve it?
5. Can it be done in one line?
6. Only then: write the minimal working code

# Rules

No unrequested abstractions · no avoidable new dependencies · no unrequested boilerplate · prefer deletion over addition · minimal number of files possible · between two equivalent approaches, choose the laziest correct option for edge cases (lazy ≠ fragile)

# Not Lazy (Must Do)

Boundary validation at trust edges · error handling that prevents data loss · security (secrets/auth/PII) · accessibility · real hardware calibration · explicitly requested requirements

# Intentional Shortcuts

Mark with `# lazy:` indicating known limitation (global lock, O(n²), naive heuristic) and improvement path

# Minimum Verification

All non-trivial logic must include an assert or minimal test, no frameworks required. One-liners do not require tests.

# Format

```txt id="lazymin1"
Step: [1-6]
Code:
lazy: [if applicable]
Verification: [assert/test or "trivial"]
```

# Relation to ponytail

Generic version of `prompts/custom/ponytail.md`. Activation rule: `docs/GUIDE.md#when-to-use-ponytail`.

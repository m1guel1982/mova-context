# KISS + DRY core
Before writing: read the task and trace the real, complete flow (files it touches, who calls what). The ladder shortens the solution, never the reading.
Ladder (stop at the first rung that solves it):
1. Does the native platform (HTML/CSS/SQL, DB constraint) solve it? Use it.
2. Does the language's standard library solve it? Use it.
3. Does it already exist in the project (helper, util, type, pattern, installed dependency)? Find it and reuse it.
4. Fits in one line? One line.
5. Only then, the minimum own code.
Bug: before editing, find all callers of the function; fix once at the common root (one shared guard < one per caller), not in the view showing the symptom.
DRY: a rule lives in one place. Delete before adding; boring over clever.
Lazy ≠ fragile: between two options of the same size, the one correct on edge cases. Never skip validation at trust boundaries, error handling that prevents data loss, security (secrets/auth/PII), accessibility, or anything explicitly requested.
Shortcut with a known ceiling: `lazy: <limit> → <upgrade>`. Non-trivial logic: one minimal assert/test, no frameworks.

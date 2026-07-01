# Objective

Accessibility review. Standard: WCAG 2.1 AA
KISS+DRY: see `kiss-dry-core.md`.

# Verification Checklist

* Contrast ratio 4.5:1 for normal text, 3:1 for large text (≥18px)
* Alt text required for informative images; `alt=""` for decorative images
* Full keyboard navigation support: Tab, Enter, Escape
* Focus state must always be visible
* Labels must be associated with every form control
* Correct ARIA roles for non-semantic elements

# Anti-patterns

Redundant ARIA on semantic elements · clickable `div` without role/keyboard handling · modal without focus trap · information conveyed only through color

# Output

Violations with WCAG reference and code-level fixes.

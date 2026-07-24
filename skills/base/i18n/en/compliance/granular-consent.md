# Skill: Granular Consent (Chile's Law 21.719 / GDPR-like regulations)
KISS+DRY: see `../engineering/kiss-dry-core.md`.

Base rule: consent must be free, specific, informed, and unambiguous. Each data-processing purpose requires its own consent — they cannot be bundled into a single "accept".

# Violation signals to look for in a flow (checkout, signup, form)

| Signal | Why it's a violation |
|---|---|
| One checkbox covering T&Cs + privacy policy + data processing | Bundles distinct purposes — the data subject can't accept one without the others |
| Checkbox pre-checked by default | Consent isn't free if it's already "accepted" |
| Transaction can't complete without accepting non-essential processing | Conditioned consent — prohibited |
| Legal text with no accessible summary | Not "informed" if the subject doesn't understand what they're accepting |
| No way to withdraw consent afterward | No ongoing control for the data subject |

# Minimum fix to propose

Split into independent checkboxes:

1. **Service T&Cs** — mandatory only if strictly required to complete the transaction
2. **Privacy policy** — informational, visible link, never blocks the action
3. **Secondary-purpose data processing** (marketing, profiling, third parties) — optional, unchecked by default, never blocks the transaction

# Reference (Chile, Law 21.719)

Fines from 5,000 to 20,000 UTM depending on severity. Applies to any data controller — regardless of size, system age, or stack.

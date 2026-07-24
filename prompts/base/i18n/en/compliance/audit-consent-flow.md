# Audit consent flow — {{PROJECT}}
Project: `{{PROJECT}}` · Module: `{{MODULE}}` · Regulation: `{{REGULATION}}`
Ockham: see `../engineering/ockham-core.md`.

# Goal
Review the flow indicated in FOCUS and determine whether it bundles distinct consent purposes into a single checkbox/action, conditions the transaction on non-essential processing, or fails to be free/specific/informed.

# Steps
1. Identify every checkbox/accept action present in the flow
2. For each one, list which purposes it bundles (T&Cs, privacy, data processing, marketing, third parties...)
3. Flag as a violation any bundling of distinct purposes into a single action
4. Propose the minimum necessary split (see the granular-consent skill)
5. Estimate regulatory risk (reference fines if applicable)

# Response format
Executive summary (2-3 lines) → list of violations (agent format) → prioritized fix checklist

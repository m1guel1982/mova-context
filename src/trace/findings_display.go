// findings_display.go — how many Security Findings actually get
// printed in a report body vs exported in full. The underlying PDF
// writer (mova.local/documents) has no page-break control - it is a
// fixed-lines-per-page text flow (see documents/pdf.go's
// linesPerPage) - so the ONLY way to bound a report to a handful of
// pages is to bound how much content is generated in the first place.
// A run with 1,500+ findings must never dump all of them into
// context-report.{md,pdf}; the full list always still exists, just in
// a separate, purpose-built file (see write_outputs.go's
// pii-audit-log.json).
package trace

import (
	"sort"

	"mova.local/sanitize"
)

// MaxPDFSecurityFindings caps how many per-file rows the Security
// Findings section of context-report.{md,pdf} ever prints. This is
// the single source both renderers (markdown_governance.go,
// pdf_governance.go) read from, so the console/markdown/PDF surfaces
// can never disagree about the cutoff.
const MaxPDFSecurityFindings = 15

// topFindingsForDisplay returns at most MaxPDFSecurityFindings
// entries from all, ranked by risk (BLOCKED first, then by
// occurrence count - the closest available proxy for "how much
// happened in this file" without a dedicated risk score), plus how
// many were left out.
func topFindingsForDisplay(all []sanitize.SecurityFinding) (top []sanitize.SecurityFinding, omitted int) {
	if len(all) <= MaxPDFSecurityFindings {
		return all, 0
	}
	ranked := make([]sanitize.SecurityFinding, len(all))
	copy(ranked, all)
	sort.SliceStable(ranked, func(i, j int) bool {
		iBlocked, jBlocked := ranked[i].Action == "BLOCKED", ranked[j].Action == "BLOCKED"
		if iBlocked != jBlocked {
			return iBlocked
		}
		return ranked[i].Occurrences > ranked[j].Occurrences
	})
	return ranked[:MaxPDFSecurityFindings], len(all) - MaxPDFSecurityFindings
}

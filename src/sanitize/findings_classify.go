// findings_classify.go — small, pure helpers EvaluateFile uses to
// turn raw detection facts into the closed "type"/"detector" slugs
// the JSON contract expects, a simple risk score, and the asset-path
// allowlist that keeps binary/vector assets out of semantic scanning
// entirely. Split out of findings.go purely to keep every file in
// this package under the 300-line limit.
package sanitize

// IsAssetPath reports whether path is a binary/visual asset
// (image/font/vector-graphic/sourcemap) that should NEVER go through
// semantic PII/secret scanning by default (see the "jina2.svg" false
// positive: an SVG's own path-data or embedded base64 easily trips
// Shannon-entropy heuristics with zero actual sensitive content).
// EvaluateFile's caller (trace/governance.go) checks this BEFORE
// calling EvaluateFile at all - an asset is simply never scanned,
// unless the person explicitly opts in (see context-trace's
// --scan-assets flag).
func IsAssetPath(path string) bool {
	lower := lowerASCII(path)
	if hasSuffixStr(lower, ".min.js") || hasSuffixStr(lower, ".min.css") {
		return true
	}
	switch extLower(path) {
	case ".svg", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".bmp", ".webp",
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		".map":
		return true
	default:
		return false
	}
}

func lowerASCII(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func hasSuffixStr(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func extLower(path string) string {
	dot := -1
	for i := len(path) - 1; i >= 0 && path[i] != '/' && path[i] != '\\'; i-- {
		if path[i] == '.' {
			dot = i
			break // the FIRST '.' found scanning backward is the real extension separator - see this fix's bug report: without this break, a multi-dot filename like "https01.drawio.svg" kept overwriting dot with the EARLIER ".drawio" position, returning ".drawio.svg" instead of ".svg" and silently defeating IsAssetPath for every multi-dot asset filename.
		}
	}
	if dot == -1 {
		return ""
	}
	out := make([]byte, len(path)-dot)
	for i, c := range []byte(path[dot:]) {
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

// classify maps raw detection facts to the small, closed "type" slug
// the JSON contract expects, plus which detector produced it.
func classify(hadPII bool, secretType string) (findingType, detector string) {
	if hadPII {
		return "pii_structural", "shannon_entropy_shape"
	}
	if secretType != "" {
		return "secret_" + slug(secretType), "regex_pattern_match"
	}
	return "", ""
}

func slug(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else if len(out) > 0 && out[len(out)-1] != '_' {
			out = append(out, '_')
		}
	}
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	return string(out)
}

// scoreFor is a deliberately simple, explainable, DETERMINISTIC risk
// heuristic — occurrence DENSITY relative to file size, capped at
// 1.0. It is not a calibrated probability; it exists so findings can
// be ranked (see MaxPDFSecurityFindings' "top by risk" ordering and
// the ConTask contract's per-finding "score") without inventing a
// black-box model. A short file with many hits scores higher than a
// huge file with the same absolute hit count.
func scoreFor(occurrences, tokens int) float64 {
	if tokens <= 0 {
		return 0
	}
	density := float64(occurrences) / float64(tokens) * 20 // scaling constant chosen so "1 hit per 20 tokens" ≈ 1.0
	if density > 1 {
		density = 1
	}
	return roundTo2(density)
}

func roundTo2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

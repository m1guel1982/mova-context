// govern.go — block-aware sanitization of the context that leaves Mova.
//
// Replaces the previous "MaskPII over the whole Focus string" approach,
// which (verified) rewrote section headers ("## FOCUS" → "[PII_…]"),
// broke code (`const aws="AKIA…"` → `const [PII_…]";`) and let full
// names and street addresses through untouched.
//
// Per block (one FOCUS:<source> file/symbol block, or memory):
//   - secrets (secrets.go shapes): BLOCK the block when the security
//     policy says so (block_on_private_key / block_on_api_key), otherwise
//     redact ONLY the secret literal, never the surrounding syntax;
//   - when PII masking is enabled:
//   - field_keys: values of structured keys ("nombre": "…") are
//     pseudonymized; the same values are masked in every other block;
//   - typed detectors (email, Chilean RUT, phone) in code and data;
//   - the structural shape/entropy scorer (MaskPII) ONLY on data blocks,
//     never on code, never on headers/markers.
//
// What this does NOT detect (documented, not hidden): names, addresses
// or other personal data in free text that never appears as a field_keys
// value and has no detectable shape.
package sanitize

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Block kinds.
const (
	KindCode = "code"
	KindData = "data"
)

var codeExts = map[string]bool{
	".go": true, ".js": true, ".mjs": true, ".cjs": true, ".ts": true, ".tsx": true, ".jsx": true,
	".py": true, ".java": true, ".kt": true, ".rb": true, ".rs": true, ".c": true, ".h": true,
	".cc": true, ".cpp": true, ".hpp": true, ".cs": true, ".php": true, ".swift": true,
	".scala": true, ".sh": true, ".ps1": true, ".sql": true, ".lua": true, ".dart": true,
	".vue": true, ".svelte": true,
}

// KindFor classifies a block by its source name ("path" or
// "path::kind=names").
func KindFor(source string) string {
	file, _, _ := strings.Cut(strings.TrimSpace(source), "::")
	if codeExts[strings.ToLower(filepath.Ext(file))] {
		return KindCode
	}
	return KindData
}

// GovernOptions configures GovernBlock / GovernBlocks.
type GovernOptions struct {
	Policy     PolicySet
	PIIEnabled bool
	// known: value → pseudonym, collected from field_keys across every
	// block of the same context (see GovernBlocks).
	known map[string]string
}

// BlockReport is the per-block evidence of what governance did.
type BlockReport struct {
	Source          string `json:"source"`
	Kind            string `json:"kind"`
	Blocked         bool   `json:"blocked,omitempty"`
	Rule            string `json:"rule,omitempty"`
	SecretsRedacted int    `json:"secrets_redacted,omitempty"`
	FieldValues     int    `json:"field_values_masked,omitempty"`
	TypedPII        int    `json:"typed_pii_masked,omitempty"`
	ShapePII        int    `json:"shape_pii_masked,omitempty"`
	KnownValues     int    `json:"known_values_masked,omitempty"`
}

// Changed reports whether governance altered or blocked the block.
func (r BlockReport) Changed() bool {
	return r.Blocked || r.SecretsRedacted+r.FieldValues+r.TypedPII+r.ShapePII+r.KnownValues > 0
}

// BlockedMarker replaces the content of a blocked block.
func BlockedMarker(rule string) string {
	return "[MOVA: contenido omitido por política " + rule + "]\n"
}

var (
	reEmail = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	reRUT   = regexp.MustCompile(`\b\d{1,2}\.?\d{3}\.?\d{3}-[\dkK]\b`)
	rePhone = regexp.MustCompile(`\+\d{1,3}[ -]?\d{1,2}[ -]?\d{3,4}[ -]?\d{4}\b`)
	// generic credential: key, separator and optional quote kept; only
	// the value (group 3) is replaced.
	reGenericCredValue = regexp.MustCompile(`(?i)(\b(?:api[_-]?key|secret|token|password|passwd)\b\s*[:=]\s*["']?)([A-Za-z0-9_\-/+]{12,})`)
	reAPIKeyValue      = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{20,}|xox[baprs]-[A-Za-z0-9-]{10,})\b`)
	reJWTValue         = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\b`)
	rePEMBlock         = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)
)

// fieldValueRe builds `"key": "value"` matcher for the policy's field keys.
func fieldValueRe(keys []string) *regexp.Regexp {
	if len(keys) == 0 {
		return nil
	}
	quoted := make([]string, 0, len(keys))
	for _, k := range keys {
		if k = strings.TrimSpace(k); k != "" {
			quoted = append(quoted, regexp.QuoteMeta(k))
		}
	}
	if len(quoted) == 0 {
		return nil
	}
	return regexp.MustCompile(`(?i)("(?:` + strings.Join(quoted, "|") + `)"\s*:\s*")([^"\n]{2,})(")`)
}

// GovernBlocks governs every block of one context with a shared
// known-values table: field_keys values found in ANY block are masked in
// ALL blocks (e.g. a name from customers.json inside a PDF's text).
func GovernBlocks(blocks []FileBlock, opt GovernOptions) ([]FileBlock, []BlockReport) {
	opt.known = map[string]string{}
	if opt.PIIEnabled {
		if re := fieldValueRe(opt.Policy.PII.FieldKeys); re != nil {
			for _, b := range blocks {
				for _, m := range re.FindAllStringSubmatch(b.Content, -1) {
					v := strings.TrimSpace(m[2])
					if len(v) >= 4 {
						opt.known[v] = pseudonym(v, opt.Policy.PII)
					}
				}
			}
		}
	}
	out := make([]FileBlock, len(blocks))
	reports := make([]BlockReport, len(blocks))
	for i, b := range blocks {
		content, rep := governBlock(b.Name, b.Content, &opt)
		out[i] = FileBlock{Name: b.Name, Content: content}
		reports[i] = rep
	}
	return out, reports
}

// GovernBlock governs a single block (e.g. a tool result) in isolation.
func GovernBlock(source, content string, opt GovernOptions) (string, BlockReport) {
	blocks, reps := GovernBlocks([]FileBlock{{Name: source, Content: content}}, opt)
	return blocks[0].Content, reps[0]
}

func governBlock(source, content string, opt *GovernOptions) (string, BlockReport) {
	rep := BlockReport{Source: source, Kind: KindFor(source)}
	sec := opt.Policy.Security

	// 1. Secrets: block or redact the literal only.
	hasPEM, hasCred := false, false
	for _, h := range DetectSecrets(content) {
		switch h.Type {
		case SecretPrivateKey:
			hasPEM = true
		case SecretAPIKey, SecretJWT, SecretGenericCred:
			hasCred = true
		}
	}
	switch {
	case hasPEM && sec.BlockOnPrivateKey:
		rep.Blocked, rep.Rule = true, "security.block_on_private_key"
		return BlockedMarker(rep.Rule), rep
	case hasCred && sec.BlockOnAPIKey:
		rep.Blocked, rep.Rule = true, "security.block_on_api_key"
		return BlockedMarker(rep.Rule), rep
	}
	content = replaceCount(content, rePEMBlock, "[REDACTED_PRIVATE_KEY]", &rep.SecretsRedacted)
	content = replaceCount(content, reAPIKeyValue, "[REDACTED_SECRET]", &rep.SecretsRedacted)
	if sec.SanitizeOnJWT {
		content = replaceCount(content, reJWTValue, "[REDACTED_SECRET]", &rep.SecretsRedacted)
	}
	if sec.SanitizeOnGenericCredential {
		content = reGenericCredValue.ReplaceAllStringFunc(content, func(m string) string {
			sub := reGenericCredValue.FindStringSubmatch(m)
			if strings.HasPrefix(sub[2], "[REDACTED") {
				return m
			}
			rep.SecretsRedacted++
			return sub[1] + "[REDACTED_SECRET]"
		})
	}
	if !opt.PIIEnabled {
		return content, rep
	}
	pol := opt.Policy.PII

	// 2. field_keys values (structured data, any kind of block).
	if re := fieldValueRe(pol.FieldKeys); re != nil {
		content = re.ReplaceAllStringFunc(content, func(m string) string {
			sub := re.FindStringSubmatch(m)
			if isAlreadyTagged(sub[2], pol) {
				return m
			}
			rep.FieldValues++
			return sub[1] + pseudonym(strings.TrimSpace(sub[2]), pol) + sub[3]
		})
	}
	// 3. known values from other blocks (longest first, literal).
	if len(opt.known) > 0 {
		vals := make([]string, 0, len(opt.known))
		for v := range opt.known {
			vals = append(vals, v)
		}
		sort.Slice(vals, func(i, j int) bool { return len(vals[i]) > len(vals[j]) })
		for _, v := range vals {
			if n := strings.Count(content, v); n > 0 {
				content = strings.ReplaceAll(content, v, opt.known[v])
				rep.KnownValues += n
			}
		}
	}
	// 4. typed detectors (deterministic regexes, code and data).
	for _, re := range []*regexp.Regexp{reEmail, reRUT, rePhone} {
		content = re.ReplaceAllStringFunc(content, func(m string) string {
			rep.TypedPII++
			return pseudonym(m, pol)
		})
	}
	// 5. structural shape/entropy scorer: data blocks only.
	if rep.Kind == KindData {
		masked, st := MaskPII(content, pol)
		content = masked
		rep.ShapePII = st.TokensMasked
	}
	return content, rep
}

func replaceCount(s string, re *regexp.Regexp, repl string, n *int) string {
	return re.ReplaceAllStringFunc(s, func(string) string {
		*n++
		return repl
	})
}

// GovernFocus applies GovernBlocks to an assembled "## FOCUS" section
// (render.go's "FOCUS:<source>" convention) plus memory, keeping the
// preamble and every marker byte-for-byte. Focus text without markers is
// governed as one data block. Returns the per-block reports.
func GovernFocus(focus, memory string, opt GovernOptions) (string, string, []BlockReport) {
	preamble, blocks := splitFocusBlocks(focus)
	unmarked := len(blocks) == 0 && strings.TrimSpace(focus) != ""
	if unmarked {
		blocks = []FileBlock{{Name: "focus", Content: focus}}
		preamble = ""
	}
	nFocus := len(blocks)
	if memory != "" {
		blocks = append(blocks, FileBlock{Name: "memory.md", Content: memory})
	}
	if len(blocks) == 0 {
		return focus, memory, nil
	}
	governed, reports := GovernBlocks(blocks, opt)
	outMemory := memory
	if memory != "" {
		outMemory = governed[nFocus].Content
	}
	switch {
	case nFocus == 0:
		return focus, outMemory, reports
	case unmarked:
		return governed[0].Content, outMemory, reports
	}
	return joinFocusBlocks(preamble, governed[:nFocus]), outMemory, reports
}

// compiler_semantic.go — Context Compiler, Fase 1: "Telegrama Semántico".
//
// Strips filler prose from instruction text (agents/skills/prompts/memory).
// Deterministic and safe: only removes low-information sentences, never
// business rules, code, or placeholders. Never call this on source code —
// it would corrupt it (see compiler_focus.go for code handling).
//
// Nothing here calls an LLM. That would add latency and cost — the exact
// opposite of the goal (fewer tokens, less latency).
package main

import (
	"regexp"
	"strings"
)

// greetingPatterns match lines that are *entirely* social noise — greetings,
// thanks, sign-offs — with no rule content. Safe to delete outright.
var greetingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*(hola|hello|hi|bienvenid[oa]s?|welcome)\b.*$`),
	regexp.MustCompile(`(?i)^\s*(gracias|thank you|thanks)\b.*$`),
	regexp.MustCompile(`(?i)^\s*(sin más preámbulos|dicho esto|having said that|that said)[,.]?\s*$`),
}

// leadinPatterns match throat-clearing prefixes at the start of a sentence.
// Only the prefix is removed — the substantive content after it is kept,
// so a critical rule phrased as "cabe destacar que X nunca debe romperse"
// survives as "X nunca debe romperse".
var leadinPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(por favor,?\s+)`),
	regexp.MustCompile(`(?i)^(please,?\s+)`),
	regexp.MustCompile(`(?i)^(cabe destacar|es importante mencionar|vale la pena mencionar)\s+que\s+`),
	regexp.MustCompile(`(?i)^(it('|)s worth (noting|mentioning)|it is important to (note|mention))\s+that\s+`),
	regexp.MustCompile(`(?i)^(como se mencionó anteriormente|as (previously\s+)?mentioned above),?\s+`),
	regexp.MustCompile(`(?i)^(en resumen|a modo de resumen|to summarize|in summary),?\s+`),
}

// criticalMarkers guard business-critical lines from ever being fully
// deleted (Fase 1 rule: "mantener intactas todas las reglas críticas").
var criticalMarkers = regexp.MustCompile(`(?i)\b(nunca|siempre|obligatori[oa]|prohibid[oa]|crític[oa]|critical|must|never|always|required|regla)\b`)

// CompileInstruction applies Telegrama Semántico to natural-language
// instruction content (agents/skills/prompts/memory). Never call this on
// source code.
func CompileInstruction(content string) string {
	if content == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	var out []string
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if strings.HasPrefix(strings.TrimSpace(trimmed), "```") {
			inFence = !inFence
			out = append(out, trimmed)
			continue
		}
		if inFence {
			out = append(out, trimmed) // never touch code fences
			continue
		}
		if isPureFiller(trimmed) {
			continue
		}
		out = append(out, collapseSpaces(stripLeadin(trimmed)))
	}
	return collapseBlankLines(strings.Join(out, "\n"))
}

// isPureFiller reports lines that are entirely social noise. Lines carrying
// a critical marker or a placeholder are never dropped.
func isPureFiller(line string) bool {
	if strings.Contains(line, "{{") || criticalMarkers.MatchString(line) {
		return false
	}
	for _, re := range greetingPatterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// stripLeadin removes a throat-clearing prefix, keeping the substance after it.
func stripLeadin(line string) string {
	leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	rest := strings.TrimLeft(line, " \t")
	for _, re := range leadinPatterns {
		if loc := re.FindStringIndex(rest); loc != nil && loc[0] == 0 {
			rest = rest[loc[1]:]
			if rest != "" {
				rest = strings.ToUpper(rest[:1]) + rest[1:]
			}
			break
		}
	}
	return leading + rest
}

var multiSpace = regexp.MustCompile(`[ \t]{2,}`)

func collapseSpaces(line string) string {
	if strings.HasPrefix(strings.TrimSpace(line), "|") {
		return line // preserve markdown tables
	}
	return multiSpace.ReplaceAllString(line, " ")
}

// collapseBlankLines removes runs of 2+ blank lines, per contexto.txt format rule.
func collapseBlankLines(s string) string {
	re := regexp.MustCompile(`\n{3,}`)
	return strings.TrimSpace(re.ReplaceAllString(s, "\n\n"))
}

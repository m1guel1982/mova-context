// highlight.go — the SINGLE language-detection implementation behind
// "cuando el chat genere código fuente, aplicar resaltado de sintaxis...
// detectar automáticamente el lenguaje" (see "4. Coloreado de código" in
// the spec this implements). AutoTagCodeFences finds ``` fences with no
// language tag and adds one, heuristically, from the code itself — the
// terminal chat's glamour renderer (cli/chat_save.go's renderMarkdown)
// then highlights the result, and MCP/HTTP responses come back as
// correctly-tagged Markdown for any client that renders it (GitHub,
// VS Code, a web UI...) — one detector, every door benefits, instead of
// three copies of "guess the language".
//
// This is deliberately a lightweight heuristic (keyword/punctuation
// scoring), not a full parser: good enough to pick the right language
// for a fence a model forgot to tag, for any of the languages Mova
// Context is expected to see (Go, Python, Java, C#, Rust, JavaScript,
// TypeScript, Kotlin, SQL, YAML, JSON, XML, Bash, and others via the
// generic fallbacks at the end).
package documents

import (
	"regexp"
	"strings"
)

// fenceLine matches an opening/closing ``` fence and captures whatever
// language tag (if any) follows it on the same line.
var fenceLine = regexp.MustCompile("^```([A-Za-z0-9_+#.-]*)\\s*$")

// AutoTagCodeFences scans text for ``` fences with no language tag and
// inserts one detected from the fenced content — fences that already
// have a tag (``` go, ```python, ...) are left untouched. Safe to call
// on any text, including text with no code fences at all (a no-op).
func AutoTagCodeFences(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inBlock := false
	openingTagged := true
	openingIdx := -1
	var blockLines []string

	for _, line := range lines {
		if !inBlock {
			if m := fenceLine.FindStringSubmatch(line); m != nil {
				inBlock = true
				openingTagged = m[1] != ""
				openingIdx = len(out)
				out = append(out, line)
				blockLines = nil
				continue
			}
			out = append(out, line)
			continue
		}

		// inBlock: a bare ``` closes it (a tagged closing fence isn't
		// standard Markdown, so any ``` here is the closing one).
		if strings.TrimSpace(line) == "```" {
			if !openingTagged {
				if lang := DetectLanguage(strings.Join(blockLines, "\n")); lang != "" {
					out[openingIdx] = "```" + lang
				}
			}
			out = append(out, blockLines...)
			out = append(out, line)
			inBlock = false
			blockLines = nil
			continue
		}
		blockLines = append(blockLines, line)
	}

	// Unterminated block (rare: a truncated response) — just emit what
	// was collected, untagged, rather than lose content.
	if inBlock {
		out = append(out, blockLines...)
	}

	return strings.Join(out, "\n")
}

// languageSignature is one language's set of recognizable, low-
// false-positive cues.
type languageSignature struct {
	name     string
	keywords []string
	patterns []*regexp.Regexp
}

var languageSignatures = []languageSignature{
	{name: "go", keywords: []string{"package ", "func ", ":=", "import (", "fmt.", "chan "}},
	{name: "python", keywords: []string{"def ", "import ", "elif ", "self.", "print(", "None", "True", "False"},
		patterns: []*regexp.Regexp{regexp.MustCompile(`(?m)^\s*#`)}},
	{name: "rust", keywords: []string{"fn ", "let mut", "impl ", "::<", "match ", "->", "pub struct"}},
	{name: "csharp", keywords: []string{"using System", "namespace ", "public class", "Console.WriteLine", "void Main"}},
	{name: "java", keywords: []string{"public class", "public static void main", "System.out.", "import java."}},
	{name: "kotlin", keywords: []string{"fun ", "val ", "var ", "companion object", "println("}},
	{name: "typescript", keywords: []string{"interface ", ": string", ": number", "export default", "import {", "as const"}},
	{name: "javascript", keywords: []string{"const ", "let ", "=>", "function ", "console.log", "require("}},
	{name: "bash", keywords: []string{"#!/bin/bash", "#!/bin/sh", "echo ", "$(", "fi\n", "then\n"}},
	{name: "sql", keywords: []string{"SELECT ", "FROM ", "WHERE ", "INSERT INTO", "CREATE TABLE", "UPDATE ", "DELETE FROM"}},
	{name: "yaml", patterns: []*regexp.Regexp{regexp.MustCompile(`(?m)^[A-Za-z_][\w-]*:\s`)}},
	{name: "json", patterns: []*regexp.Regexp{regexp.MustCompile(`(?s)^\s*[\{\[].*[\}\]]\s*$`)}},
	{name: "xml", patterns: []*regexp.Regexp{regexp.MustCompile(`(?s)^\s*<\?xml|^\s*<[A-Za-z][\w:-]*[\s>]`)}},
	{name: "html", keywords: []string{"<!DOCTYPE html", "<html", "<div", "<body"}},
}

// DetectLanguage guesses the language of a fenced code block from its
// content — keyword/pattern scoring, highest score wins; empty string
// when nothing scores (glamour/most renderers still highlight generic
// syntax reasonably in that case).
func DetectLanguage(code string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return ""
	}

	best := ""
	bestScore := 0
	for _, sig := range languageSignatures {
		score := 0
		for _, kw := range sig.keywords {
			if strings.Contains(code, kw) {
				score++
			}
		}
		for _, p := range sig.patterns {
			if p.MatchString(trimmed) {
				score += 2
			}
		}
		if score > bestScore {
			bestScore = score
			best = sig.name
		}
	}
	return best
}

package applyflow

import (
	"strings"
	"time"

	"mova.local/documents"
	"mova.local/i18n"
	"mova.local/patcher"
)

// Result de aplicar una selección.
type Result struct {
	Applied []patcher.AppliedFile
	Skipped []patcher.SkippedBlock
	Text    string // resumen ya traducido, listo para mostrar
}

// Apply escribe SOLO los cambios seleccionados (índices 1..N) y devuelve el
// resumen. Es la única función que toca el disco en todo el flujo.
func (p *Proposal) Apply(root string, sel map[int]bool) Result {
	t := func(k string, a ...map[string]any) string { return i18n.TIn(p.Lang, k, a...) }
	var blocks []documents.LabeledCodeBlock
	for _, c := range p.Changes {
		if sel[c.Index] {
			blocks = append(blocks, c.block())
		}
	}
	opt := p.options()
	opt.Backup = p.Backup
	opt.Stamp = time.Now().Format("20060102-150405")
	applied, skipped, _ := patcher.ApplyBlocksOpts(p.Repo, blocks, opt)

	var b strings.Builder
	for _, a := range applied {
		target := a.Path
		if a.Symbol != "" {
			target += "::" + a.Symbol + "()"
		}
		b.WriteString(t("apply.applied", map[string]any{"target": target}) + "\n")
		if a.Backup != "" {
			b.WriteString(t("apply.backup", map[string]any{"path": a.Backup}) + "\n")
		}
	}
	for _, s := range skipped {
		target := s.Path
		if s.Symbol != "" {
			target += "::" + s.Symbol + "()"
		}
		b.WriteString(t("apply.skipped", map[string]any{"target": target, "reason": s.Reason}) + "\n")
	}
	b.WriteString(t("apply.summary", map[string]any{"applied": len(applied), "skipped": len(skipped)}))
	return Result{Applied: applied, Skipped: skipped, Text: b.String()}
}

// Answer aplica la respuesta de la persona a la propuesta: devuelve el
// texto a mostrar y si se escribió algo. Respuesta inválida o "no" = no
// toca nada.
func (p *Proposal) Answer(root, answer string) (text string, wrote bool) {
	t := func(k string, a ...map[string]any) string { return i18n.TIn(p.Lang, k, a...) }
	sel, kind := ParseSelection(answer, len(p.Changes))
	switch kind {
	case AnswerNone:
		return t("apply.cancelled"), false
	case AnswerInvalid:
		return t("apply.invalid", map[string]any{"count": len(p.Changes)}), false
	}
	res := p.Apply(root, sel)
	return res.Text, len(res.Applied) > 0
}

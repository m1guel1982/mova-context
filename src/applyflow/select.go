package applyflow

import (
	"strconv"
	"strings"
)

// Answer es la interpretación de la respuesta a la pregunta.
type Answer int

const (
	AnswerInvalid Answer = iota
	AnswerAll
	AnswerSome
	AnswerNone
)

// ParseSelection interpreta la respuesta: "s/sí/y/yes/a/todos" = todos;
// "1,3" o "#1,#3" o "1 3" = solo esos; "n/no/c/cancelar/ninguno" o vacío =
// ninguno. Cualquier otra cosa es inválida y NO modifica nada.
func ParseSelection(answer string, n int) (map[int]bool, Answer) {
	sel := map[int]bool{}
	a := strings.ToLower(strings.TrimSpace(answer))
	switch a {
	case "s", "sí", "si", "y", "yes", "a", "all", "todo", "todos", "todas":
		for i := 1; i <= n; i++ {
			sel[i] = true
		}
		return sel, AnswerAll
	case "", "n", "no", "c", "cancel", "cancelar", "ninguno", "ninguna", "none":
		return sel, AnswerNone
	}
	fields := strings.FieldsFunc(a, func(r rune) bool { return r == ',' || r == ' ' || r == ';' })
	for _, f := range fields {
		v, err := strconv.Atoi(strings.TrimPrefix(f, "#"))
		if err != nil || v < 1 || v > n {
			return map[int]bool{}, AnswerInvalid
		}
		sel[v] = true
	}
	if len(sel) == 0 {
		return sel, AnswerInvalid
	}
	return sel, AnswerSome
}

// LooksLikeAnswer: ¿el mensaje parece una respuesta a la pregunta (y no una
// consulta nueva)? Se usa en MCP/HTTP para no secuestrar mensajes normales
// cuando hay una propuesta pendiente.
func LooksLikeAnswer(message string, n int) bool {
	_, ans := ParseSelection(message, n)
	return ans != AnswerInvalid && len(strings.Fields(message)) <= 6
}

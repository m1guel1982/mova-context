// match.go — "LIKE simple" text matching (edición Community, gratis).
//
// Antes de este archivo, cada resolver de texto (Markdown, Legal, Memory,
// Fallback) comparaba con `strings.Contains(strings.ToLower(line), q)`:
// coincidencia EXACTA de substring, sensible a acentos. En la práctica
// esto hacía que `focus` fuera muy rígido — ver el ejemplo real de este
// mismo repo (projects/i18n-demo): buscar "articulo 3" (sin tilde, como
// escribe casi todo el mundo al tipear rápido) no encontraba
// "## Artículo 3 — Pluralización" porque "articulo" ≠ "artículo" para un
// Contains estricto.
//
// LikeContains implementa el equivalente de un `WHERE texto ILIKE
// '%query%'` de SQL: insensible a mayúsculas/minúsculas Y a acentos, sin
// heurísticas probabilísticas, sin LLM, 100% determinista — el mismo
// input siempre da el mismo resultado. Esto es TODO lo que la edición
// Community promete (y es exactamente lo que se pidió): amigable, nunca
// frustra con un "0 resultados" por una tilde, pero sigue siendo búsqueda
// de texto, no de significado.
//
// Lo que esto NO hace (y por diseño, nunca hará en Community): entender
// que "LoadTranslations" está relacionado con "Carga de archivos de
// idiomas" o "i18n" cuando esas palabras no aparecen en el texto. Eso
// requiere embeddings — ver mova.local/compiler/focus.SemanticResolver
// (edición Premium), que se registra ANTES que los resolvers de este
// paquete y solo entra en juego cuando el proyecto configura
// project.json "embedding" y el binario tiene el tag "premium".
package focus

import "strings"

// accentFolds mapea cada vocal acentuada/con diéresis (y ñ) a su
// equivalente simple. Cubre español (el idioma de casi todo el corpus de
// ejemplo de este repo) e ignora silenciosamente cualquier otro rune —
// nunca produce un caracter inválido, en el peor caso no lo normaliza.
var accentFolds = map[rune]rune{
	'á': 'a', 'à': 'a', 'ä': 'a', 'â': 'a', 'Á': 'a', 'À': 'a', 'Ä': 'a', 'Â': 'a',
	'é': 'e', 'è': 'e', 'ë': 'e', 'ê': 'e', 'É': 'e', 'È': 'e', 'Ë': 'e', 'Ê': 'e',
	'í': 'i', 'ì': 'i', 'ï': 'i', 'î': 'i', 'Í': 'i', 'Ì': 'i', 'Ï': 'i', 'Î': 'i',
	'ó': 'o', 'ò': 'o', 'ö': 'o', 'ô': 'o', 'Ó': 'o', 'Ò': 'o', 'Ö': 'o', 'Ô': 'o',
	'ú': 'u', 'ù': 'u', 'ü': 'u', 'û': 'u', 'Ú': 'u', 'Ù': 'u', 'Ü': 'u', 'Û': 'u',
	'ñ': 'n', 'Ñ': 'n',
	'ç': 'c', 'Ç': 'c',
}

// foldAccents quita acentos/diéresis rune por rune — sin depender de
// paquetes externos de normalización Unicode (Go stdlib no trae uno, y
// este proyecto prefiere cero dependencias nuevas para algo tan acotado:
// el alfabeto de los idiomas que hoy soporta patterns/*.json, es/en).
func foldAccents(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if folded, ok := accentFolds[r]; ok {
			b.WriteRune(folded)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// normalizeForMatch — minúsculas + sin acentos. Es la ÚNICA transformación
// que aplican los resolvers de texto para decidir si algo "matchea": nunca
// se reordenan palabras, nunca se corrigen errores de tipeo más allá de
// acentos, nunca se interpretan sinónimos (eso es Premium).
func normalizeForMatch(s string) string {
	return foldAccents(strings.ToLower(s))
}

// LikeContains reporta si needle aparece dentro de haystack como
// substring, ignorando mayúsculas/minúsculas y acentos — el equivalente
// determinista de `ILIKE '%needle%'`. needle vacío siempre matchea
// (mismo comportamiento que strings.Contains).
func LikeContains(haystack, needle string) bool {
	return strings.Contains(normalizeForMatch(haystack), normalizeForMatch(needle))
}

// ExactMarker prefixes a `focus` item to request EXACT matching instead of
// the default "LIKE simple": "=Órdenes" matches only the literal text
// "Órdenes" (same case, same accents) — never "ordenes", never "ÓRDENES".
// Symmetric with the existing "()" suffix that marks a code symbol; the
// two can be combined ("=CreateOrder()" — exact code symbol, no LIKE
// fallback pass). Absent marker = current default behavior (LIKE).
const ExactMarker = "="

// IsExact reports whether a raw `focus` item (as written in project.json,
// before any other stripping) requests exact matching.
func IsExact(target string) bool {
	return strings.HasPrefix(target, ExactMarker)
}

// StripExact removes the leading "=" marker, if present. Safe to call on
// any target, exact or not.
func StripExact(target string) string {
	return strings.TrimPrefix(target, ExactMarker)
}

// LikeExact is the "=" exact mode: case- and accent-SENSITIVE substring
// match — plain byte-for-byte strings.Contains, zero folding. This is the
// equivalent of SQL's `WHERE texto LIKE '%needle%' COLLATE "C"` (a
// case-sensitive collation) — still a substring match, but nothing is
// normalized away, so "Órdenes" and "órdenes" are different strings.
func LikeExact(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// LikeContainsAllWords reporta si CADA palabra de needle aparece en algún
// lugar de haystack (en cualquier orden, no necesariamente juntas) —
// usado como último recurso, más tolerante, por FallbackResolver cuando
// ni la ruta exacta ni ninguna sección estructurada calzó. Sigue siendo
// 100% textual: exige que las palabras EXISTAN en el texto, nunca infiere
// palabras relacionadas que no están escritas.
//
// needle con una sola palabra se comporta igual que LikeContains.
func LikeContainsAllWords(haystack, needle string) bool {
	words := strings.Fields(normalizeForMatch(needle))
	if len(words) == 0 {
		return true
	}
	h := normalizeForMatch(haystack)
	for _, w := range words {
		if !strings.Contains(h, w) {
			return false
		}
	}
	return true
}

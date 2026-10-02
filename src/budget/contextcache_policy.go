// contextcache_policy.go — cuándo se puede usar mova-context-cache.json.
//
// El caché guarda el texto de Focus y Memory YA saneado pero ANTES del
// enmascarado de PII (BuildGatedContext sanea → cachea → recién entonces
// enmascara). Con pii_masking activo, ese archivo dejaría en disco justo
// el texto que el enmascarado debía proteger. Por eso, con PII activo el
// caché se desactiva y se borra el que hubiera de ejecuciones anteriores:
// se pierde solo el ahorro de CPU, nunca el aislamiento de datos.
package budget

import (
	"os"

	"mova.local/core"
)

func useContextCache(root, project string, cfg *core.BudgetConfig) bool {
	if !core.ContextCacheEnabled(cfg) {
		return false
	}
	if core.PIIMaskingEnabled(cfg) {
		_ = os.Remove(ContextCachePath(root, project))
		return false
	}
	return true
}

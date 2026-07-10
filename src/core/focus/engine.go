// Package focus implements the Focus Resolution Engine — Community
// edition. Historia: nació dentro del módulo comercial
// (mova.local/compiler/focus), disponible SOLO en binarios compilados con
// "-tags premium". Se movió a mova.local/core/focus (este paquete) para
// que la funcionalidad BÁSICA de `focus` — resolución por ruta exacta,
// símbolo de código, sección de documento, tabla SQL, nodo JSON, y una
// búsqueda de texto "LIKE" tolerante a mayúsculas/acentos — esté
// disponible de forma GRATUITA en la edición Community, sin ningún build
// tag. Ver docs/i18n/{es,en}/focus-engine.md para la explicación completa
// de qué es gratis y qué sigue siendo de pago.
//
// Lo que SÍ sigue siendo exclusivo de la edición Premium/Enterprise:
//   - Distilación de texto (Semantic Cleanup, Fase 1 del Compiler v2).
//   - El pipeline completo (Normalization, Token Optimization, Priority
//     Ranking, Budget Assembly).
//   - "Like Semántico": un SemanticResolver adicional (embeddings +
//     pgvector) que se registra ANTES que los resolvers de este paquete
//     — ver mova.local/compiler/focus (módulo comercial). Cuando el
//     SemanticResolver no encuentra nada por encima de su umbral de
//     confianza (o el proyecto no tiene "embedding" configurado, o el
//     binario es Community), la cascada sigue exactamente con los
//     resolvers de ESTE paquete — la búsqueda "LIKE simple" nunca deja
//     de funcionar como red de seguridad, en ninguna edición.
//
// Principio rector (sin cambios respecto al diseño original):
//
//	"El Focus Engine no resuelve archivos; resuelve unidades de conocimiento."
//
// El motor nunca decide POR SÍ MISMO cómo extraer una unidad de conocimiento.
// Solo orquesta: recibe un target (string tal como aparece en project.json
// "focus"), lo ofrece a cada Resolver registrado en orden de prioridad, y
// devuelve el primer resultado no vacío. Cada Resolver es responsable de un
// único tipo de conocimiento (archivo, directorio, símbolo de código, sección
// de documento, bloque cronológico, nodo JSON, definición SQL, ...).
//
// Determinismo (obligatorio):
//   - nunca usa un LLM
//   - nunca usa heurísticas probabilísticas
//   - nunca depende del orden del sistema operativo (los resolvers que
//     recorren el filesystem ordenan explícitamente antes de decidir)
//   - mismo input (mismo repo + mismo target) → mismo output
package focus

import "fmt"

// Context es el entorno de resolución compartido por todos los resolvers:
// la raíz del repo contra la que se resuelven rutas relativas, más (desde
// la transparencia de reportes, ver focus/stats.go) qué carpetas ignorar
// y dónde acumular evidencia real de qué tocó el escaneo. ExcludeDirs y
// Stats son opcionales: un Context{RepoPath: x} vacío sigue funcionando
// exactamente igual que antes — solo no acumula estadísticas.
type Context struct {
	RepoPath string

	// ExcludeDirs: carpetas adicionales (además de las que fsutil.go
	// siempre ignora — .git, node_modules, vendor, dist, build,
	// __pycache__, .venv, venv, .idea, .vscode) que ningún resolver debe
	// descender. Viene de project.json (contextCompiler.focus_exclude).
	ExcludeDirs []string

	// Stats acumula, de forma compartida entre TODOS los resolvers y
	// TODOS los targets de un mismo `mova compile` (o `mova run`),
	// cuántos archivos vio el escaneo del repo y por qué el resto no
	// entró — evidencia real para contexto.report, nunca una
	// estimación. nil es válido: los métodos de abajo son no-op sin
	// panic cuando Stats es nil.
	Stats *ScanStats
}

// ContextBlock es la unidad mínima de conocimiento que produce un resolver.
// Kind identifica el tipo de resolución para que la capa de presentación
// sepa cómo formatearlo sin necesidad de que el motor conozca nada sobre
// formato de salida.
type ContextBlock struct {
	Source  string // ruta relativa al repo, o vacío si no aplica
	Kind    string // "file" | "dir-index" | "code-symbol" | "doc-section" | "chronological" | "json-node" | "sql-def" | "bounded-excerpt" | "semantic" (premium)
	Content string
}

// Resolver es el contrato único que implementa cada tipo de resolución.
// Match debe ser barato (sin I/O costoso cuando sea posible) — decide si
// ESTE resolver es candidato para el target. Resolve hace el trabajo real
// y puede fallar (target candidato pero sin resultado, p. ej. un symbol
// que no aparece en ningún archivo del repo).
type Resolver interface {
	Match(ctx Context, target string) bool
	Resolve(ctx Context, target string) ([]ContextBlock, error)
}

// Engine consulta los resolvers registrados en orden de prioridad.
//
// A diferencia de un simple "primer Match gana", Resolve avanza en cascada:
// si un resolver hace Match pero su Resolve no encuentra nada (ErrNotFound),
// el motor continúa con el siguiente resolver en la lista. Esto es lo que
// permite que la edición Premium anteponga un SemanticResolver (embeddings)
// sin duplicar ni reemplazar los resolvers "LIKE" de este paquete: si el
// SemanticResolver no está seguro, la cascada sigue exactamente igual que
// en Community.
type Engine struct {
	resolvers []Resolver
}

// ErrNotFound señala que un resolver era candidato (Match == true) pero no
// produjo ningún ContextBlock. No es un error de sistema — es información
// para que el motor continúe la cascada.
var ErrNotFound = fmt.Errorf("focus: target not found by this resolver")

// New crea un motor vacío. Sin resolvers registrados, Resolve siempre falla.
func New() *Engine {
	return &Engine{}
}

// RegisterResolver añade un resolver al final de la cadena de prioridad.
// El orden de registro ES el orden de prioridad — quien construye el motor
// decide qué tipos de conocimiento se intentan primero. Añadir un nuevo tipo
// de conocimiento (un nuevo adapter, un nuevo formato de documento, o el
// SemanticResolver de la edición Premium) nunca requiere modificar el
// Engine ni los resolvers existentes (Open/Closed).
func (e *Engine) RegisterResolver(r Resolver) {
	e.resolvers = append(e.resolvers, r)
}

// Resolve intenta cada resolver registrado, en orden, hasta que uno produzca
// al menos un ContextBlock. Devuelve error solo si ningún resolver pudo
// resolver el target.
func (e *Engine) Resolve(ctx Context, target string) ([]ContextBlock, error) {
	for _, r := range e.resolvers {
		if !r.Match(ctx, target) {
			continue
		}
		blocks, err := r.Resolve(ctx, target)
		if err == nil && len(blocks) > 0 {
			return blocks, nil
		}
		// Match == true pero sin resultado: no es un fallo del motor,
		// es una señal para probar el siguiente resolver de la cascada.
	}
	return nil, fmt.Errorf("focus: no resolver matched target: %s", target)
}

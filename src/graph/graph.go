// Package graph genera, sin LLM y de forma determinista, el grafo de
// dependencias AST de lo que cada tarea declara en "focus"/"exclude"
// (llamadas, referencias a variables e importaciones entre archivos y
// símbolos) y lo escribe como .png/.svg/.pdf. Se genera un grafo por cada
// tarea que traiga "graph" (ver core.Task.GraphTarget), no solo el de la
// activa. Se engancha al motor por core.GraphHook (ver core/graph_hook.go
// para el porqué) y, en chat/MCP/HTTP, trabaja en segundo plano
// (core.GraphAsync) para que nada espere un render.
package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mova.local/core"
	"mova.local/diagram"
	"mova.local/documents"
	"mova.local/i18n"
)

func init() { core.GraphHook = generate }

// tr traduce con el "lang" del proyecto (project.json) y, si no hay o no
// existe, con el idioma global de config/lang/lang_active.json.
func tr(lang, key string, args ...map[string]any) string { return i18n.TIn(lang, key, args...) }

// revalidate: pasado este tiempo se vuelve a resolver focus/exclude a
// archivos (por si apareció uno nuevo que calza con un nombre suelto).
const revalidate = 20 * time.Second

type stamp struct {
	path        string
	size, mtime int64
}

// entry recuerda qué se generó y con qué: la huella del pedido (project.json:
// focus/exclude/graph/lang…) y tamaño+mtime de cada archivo analizado.
type entry struct {
	fp      string
	stamps  []stamp
	status  string
	checked time.Time
	projDir string // dueño en disco (graph-cache.json); "" si viene de disco aún sin reclamar
}

var (
	mu       sync.Mutex
	cache    = map[string]*entry{} // por archivo de salida
	latest   = map[string]string{} // huella más reciente agendada por salida
	inflight = map[string]string{} // huella en ejecución por salida
	slots    = make(chan struct{}, 2)
)

// diskFile: la caché sobrevive entre procesos (cada `mova chat` es uno nuevo):
// con las fuentes y project.json sin cambios, un arranque NO vuelve a
// renderizar. Un solo archivo por proyecto, junto a project.json.
const diskFile = "graph-cache.json"

type diskEntry struct {
	FP     string  `json:"fp"`
	Stamps []stamp `json:"stamps"`
	Status string  `json:"status"`
}

var loaded = map[string]bool{} // carpetas de proyecto cuya caché en disco ya se leyó

func (s stamp) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{s.path, s.size, s.mtime})
}

func (s *stamp) UnmarshalJSON(b []byte) error {
	var v []json.RawMessage
	if err := json.Unmarshal(b, &v); err != nil || len(v) != 3 {
		return fmt.Errorf("graph: bad cache stamp")
	}
	if err := json.Unmarshal(v[0], &s.path); err != nil {
		return err
	}
	if err := json.Unmarshal(v[1], &s.size); err != nil {
		return err
	}
	return json.Unmarshal(v[2], &s.mtime)
}

// loadDisk lee la caché de la carpeta una sola vez por proceso. Las entradas
// quedan con checked en cero: la primera consulta las revalida (solo resolver
// + stat), sin renderizar.
func loadDisk(projDir string) {
	mu.Lock()
	defer mu.Unlock()
	if loaded[projDir] {
		return
	}
	loaded[projDir] = true
	data, err := os.ReadFile(filepath.Join(projDir, diskFile))
	if err != nil {
		return
	}
	var m map[string]diskEntry
	if json.Unmarshal(data, &m) != nil {
		return
	}
	for out, d := range m {
		if _, ok := cache[out]; !ok {
			cache[out] = &entry{fp: d.FP, stamps: d.Stamps, status: d.Status, projDir: projDir}
		}
	}
}

// saveDisk escribe (atómico) las entradas de esta carpeta de proyecto.
func saveDisk(projDir string) {
	mu.Lock()
	m := map[string]diskEntry{}
	for out, e := range cache {
		if e.projDir == projDir && !e.checked.IsZero() { // las de disco nunca revalidadas (tarea borrada) se descartan
			m[out] = diskEntry{e.fp, e.stamps, e.status}
		}
	}
	mu.Unlock()
	data, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return
	}
	path := filepath.Join(projDir, diskFile)
	tmp := fmt.Sprintf("%s.tmp-%d", path, os.Getpid())
	if os.MkdirAll(projDir, 0o755) == nil && os.WriteFile(tmp, data, 0o644) == nil {
		if os.Rename(tmp, path) != nil {
			_ = os.Remove(path)
			if os.Rename(tmp, path) != nil {
				_ = os.Remove(tmp)
			}
		}
	}
}

// generate devuelve una línea de estado por tarea. Camino rápido: si la huella
// del pedido no cambió y los archivos analizados siguen iguales (solo
// `stat`), responde desde la caché sin resolver ni parsear nada — el chat
// reconstruye el contexto en cada turno, así que esto tiene que ser casi gratis.
func generate(req core.GraphRequest) string {
	projDir := filepath.Dir(core.ProjectJSONPath(req.Root, req.ProjectName))
	loadDisk(projDir)
	results := make([]string, len(req.Jobs))
	used := map[string]string{}
	var wg sync.WaitGroup
	for i, job := range req.Jobs {
		out, errMsg := resolveOutput(projDir, job.OutSpec, req.Lang, job.Task)
		if errMsg != "" {
			results[i] = errMsg
			continue
		}
		if other, dup := used[out]; dup {
			results[i] = tr(req.Lang, "graph.dup_output", map[string]any{"task": job.Task, "other": other, "path": out})
			continue
		}
		used[out] = job.Task
		sp := spec{RepoPath: req.RepoPath, ProjectName: req.ProjectName, TaskName: job.Task, Lang: req.Lang, Focus: job.Focus, Exclude: job.Exclude}
		fp := fingerprint(sp, job.OutSpec, out)
		if st, ok := fresh(out, fp); ok {
			results[i] = st
			continue
		}
		if core.GraphAsync {
			results[i] = schedule(sp, projDir, out, fp)
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], _ = render(sp, projDir, out, fp)
		}(i)
	}
	wg.Wait()
	var lines []string
	for _, r := range results {
		if r != "" {
			lines = append(lines, r)
		}
	}
	return strings.Join(lines, "\n")
}

// fresh: ¿lo ya generado sigue valiendo? Solo `stat` de los archivos.
func fresh(out, fp string) (string, bool) {
	mu.Lock()
	e := cache[out]
	mu.Unlock()
	if e == nil || e.fp != fp || time.Since(e.checked) > revalidate || !stampsEqual(e.stamps) {
		return "", false
	}
	if _, err := os.Stat(out); err != nil {
		return "", false
	}
	return e.status, true
}

// schedule agenda el render en segundo plano y vuelve al instante. Si ya hay
// algo generado para esta salida lo devuelve mientras se revalida; si no,
// avisa que se está generando (al terminar llega por core.GraphNotify).
func schedule(sp spec, projDir, out, fp string) string {
	mu.Lock()
	e := cache[out]
	latest[out] = fp
	if inflight[out] == fp {
		mu.Unlock()
		return pending(sp, out, e, fp)
	}
	inflight[out] = fp
	mu.Unlock()

	go func() {
		slots <- struct{}{}
		defer func() { <-slots }()
		mu.Lock()
		superseded := latest[out] != fp // llegó un project.json más nuevo mientras esperaba turno
		mu.Unlock()
		var status string
		var regenerated bool
		if !superseded {
			status, regenerated = render(sp, projDir, out, fp)
		}
		mu.Lock()
		if inflight[out] == fp {
			delete(inflight, out)
		}
		mu.Unlock()
		if regenerated && core.GraphNotify != nil {
			core.GraphNotify(status)
		}
	}()
	return pending(sp, out, e, fp)
}

func pending(sp spec, out string, e *entry, fp string) string {
	if e != nil && e.fp == fp {
		return e.status
	}
	return tr(sp.Lang, "graph.generating", map[string]any{"task": sp.TaskName, "path": out})
}

// render hace el trabajo: resuelve archivos, revalida contra la caché y, solo
// si algo cambió, analiza, dibuja y escribe (atómico). regenerated indica que
// se escribió un archivo nuevo.
func render(sp spec, projDir, out, fp string) (status string, regenerated bool) {
	files := locate(sp)
	stamps := stampsOf(files)
	mu.Lock()
	e := cache[out]
	mu.Unlock()
	if e != nil && e.fp == fp && stampsSame(e.stamps, stamps) {
		if _, err := os.Stat(out); err == nil {
			mu.Lock()
			claimed := e.checked.IsZero()
			e.checked, e.projDir = time.Now(), projDir
			mu.Unlock()
			if claimed {
				saveDisk(projDir)
			}
			return e.status, false
		}
	}

	g, st := build(sp, files)
	args := map[string]any{"task": sp.TaskName}
	if len(g.Nodes) == 0 {
		status = tr(sp.Lang, "graph.empty", args)
	} else {
		if err := diagram.ExportGraph(g, out); err != nil {
			args["err"] = err.Error()
			return tr(sp.Lang, "graph.err_write", args), false
		}
		args["path"], args["files"], args["nodes"], args["edges"] = out, len(g.Clusters), len(g.Nodes), len(g.Edges)
		status = tr(sp.Lang, "graph.generated", args)
		regenerated = true
	}
	for _, w := range st.warnings {
		status += "\n" + w
	}
	mu.Lock()
	cache[out] = &entry{fp: fp, stamps: stamps, status: status, checked: time.Now(), projDir: projDir}
	mu.Unlock()
	saveDisk(projDir)
	return status, regenerated
}

func fingerprint(sp spec, outSpec, out string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s|%v|%v|%s", out, sp.Lang, sp.RepoPath, sp.ProjectName, sp.TaskName, sp.Focus, sp.Exclude, outSpec)
	return hex.EncodeToString(h.Sum(nil))
}

func stampsOf(files []located) []stamp {
	out := make([]stamp, 0, len(files))
	for _, f := range files {
		if fi, err := os.Stat(f.path); err == nil {
			out = append(out, stamp{f.path, fi.Size(), fi.ModTime().UnixNano()})
		}
	}
	return out
}

func stampsSame(a, b []stamp) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func stampsEqual(st []stamp) bool {
	for _, s := range st {
		if fi, err := os.Stat(s.path); err != nil || fi.Size() != s.size || fi.ModTime().UnixNano() != s.mtime {
			return false
		}
	}
	return true
}

// resolveOutput convierte el valor de "graph" en la ruta real de salida.
// Relativa -> carpeta de project.json; absoluta/UNC -> mismo criterio
// multiplataforma que el resto de Mova (documents.NormalizeAbsPath, que
// rechaza con un error claro un formato que este SO no puede satisfacer).
// Sin extensión -> .png; un directorio -> graph.png dentro.
func resolveOutput(projDir, spec, lang, task string) (path, errMsg string) {
	spec = strings.TrimSpace(spec)
	dirLike := strings.HasSuffix(spec, "/") || strings.HasSuffix(spec, `\`)
	if documents.IsAbsCrossPlatform(spec) {
		n, err := documents.NormalizeAbsPath(spec)
		if err != nil {
			return "", tr(lang, "graph.err_path", map[string]any{"task": task, "err": err.Error()})
		}
		path = n
	} else {
		path = filepath.Join(projDir, filepath.FromSlash(strings.ReplaceAll(spec, `\`, "/")))
	}
	if fi, err := os.Stat(path); dirLike || (err == nil && fi.IsDir()) {
		path = filepath.Join(path, core.DefaultGraphFile)
	}
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".png", ".svg", ".pdf":
	case "":
		path += ".png"
	default:
		return "", tr(lang, "graph.err_format", map[string]any{"task": task, "ext": ext})
	}
	return path, ""
}

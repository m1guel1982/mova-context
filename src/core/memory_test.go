package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const memBlock = "```memory\n## YYYY-MM-DD — session\n**Tarea:** analizar\n**Realizado:** análisis de ETA\n**Hallazgos:** `planner.js::_mapOrderHistory` — FIELD_B — se pierde — _processOrder reconstruye el objeto\n**Datos clave:** FIELD_D, FIELD_B, OFB_ACT, TKO_ACT\n```"

func writeProj(t *testing.T, root, extra string) {
	t.Helper()
	dir := filepath.Join(root, "projects", "p")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"project":"p","repo":".","lang":"es","tasks":{"analizar":{"prompt":"x"},"agregar-columnas":{"prompt":"y"}}` + extra + `}`
	if err := os.WriteFile(filepath.Join(dir, "project.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

func longReply(block string) string {
	return "Informe técnico del análisis. " + strings.Repeat("Detalle del recorrido ETA. ", 12) + "\n\n" + block
}

func TestMemorySettingValues(t *testing.T) {
	cases := []struct {
		raw  string
		on   bool
		path string
	}{
		{``, false, ""}, {`null`, false, ""}, {`false`, false, ""}, {`true`, true, ""},
		{`""`, false, ""}, {`"false"`, false, ""}, {`"true"`, true, ""}, {`"si"`, true, ""},
		{`"D:\\mova\\memoria"`, true, `D:\mova\memoria`}, {`"/mnt/data/mem.md"`, true, "/mnt/data/mem.md"},
		{`123`, false, ""},
	}
	for _, c := range cases {
		p := &Project{Memory: json.RawMessage(c.raw)}
		on, path := p.MemorySetting()
		if on != c.on || path != c.path {
			t.Errorf("memory=%s → (%v,%q), want (%v,%q)", c.raw, on, path, c.on, c.path)
		}
	}
}

func TestMemoryPathResolution(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(t.TempDir(), "externa")
	cases := []struct{ name, extra, want string }{
		{"ausente", ``, filepath.Join(root, "projects", "p", "memory.md")},
		{"true", `,"memory":true`, filepath.Join(root, "projects", "p", "memory.md")},
		{"relativa", `,"memory":"mem/m.md"`, filepath.Join(root, "projects", "p", "mem", "m.md")},
		{"carpeta con barra", `,"memory":"mem/"`, filepath.Join(root, "projects", "p", "mem", "memory.md")},
		{"absoluta archivo", fmt.Sprintf(`,"memory":%q`, filepath.Join(abs, "m.md")), filepath.Join(abs, "m.md")},
		{"absoluta carpeta", fmt.Sprintf(`,"memory":%q`, abs), filepath.Join(abs, "memory.md")},
		{"memory_path histórico", `,"memory_path":"viejo/memory.md"`, filepath.Join(root, "viejo", "memory.md")},
		{"memory gana a memory_path", `,"memory":"nuevo/","memory_path":"viejo/memory.md"`, filepath.Join(root, "projects", "p", "nuevo", "memory.md")},
	}
	for _, c := range cases {
		writeProj(t, root, c.extra)
		got, err := ResolveMemoryTarget(root, "p")
		if err != nil || got != c.want {
			t.Errorf("%s: got (%q,%v), want %q", c.name, got, err, c.want)
		}
	}
	// Ruta de Windows en un servidor Linux/macOS: error claro, no un archivo fantasma.
	writeProj(t, root, `,"memory":"C:\\mova\\memory.md"`)
	if _, err := ResolveMemoryTarget(root, "p"); err == nil && filepath.Separator == '/' {
		t.Errorf("una ruta C:\\ en Linux debe dar error explícito")
	}
}

func TestRecordMemoryOffUnlessEnabled(t *testing.T) {
	for _, extra := range []string{``, `,"memory":false`, `,"memory":"false"`} {
		root := t.TempDir()
		writeProj(t, root, extra)
		res, err := RecordMemory(NewFileAdapter(root), root, "p", "analizar", longReply(memBlock), RecordOptions{})
		if err != nil || !res.Off || res.Saved != 0 {
			t.Fatalf("%q: want Off, got %+v err=%v", extra, res, err)
		}
		if _, err := os.Stat(MemoryPath(root, "p")); err == nil {
			t.Fatalf("%q: memory.md must NOT exist when memory is off", extra)
		}
	}
	root := t.TempDir()
	writeProj(t, root, `,"memory":true`)
	res, err := RecordMemory(NewFileAdapter(root), root, "p", "analizar", longReply(memBlock), RecordOptions{})
	if err != nil || res.Saved != 1 {
		t.Fatalf("memory:true must record: %+v err=%v", res, err)
	}
	// Force (acción explícita /memory) escribe aunque el campo esté apagado.
	root2 := t.TempDir()
	writeProj(t, root2, ``)
	if r, _ := RecordMemory(NewFileAdapter(root2), root2, "p", "", memBlock, RecordOptions{Force: true}); r.Saved != 1 {
		t.Fatalf("Force must write: %+v", r)
	}
}

func TestRecordMemoryFormatDedupeAndBlocks(t *testing.T) {
	root := t.TempDir()
	writeProj(t, root, `,"memory":true`)
	ad := NewFileAdapter(root)
	reply := longReply(memBlock)
	if r, _ := RecordMemory(ad, root, "p", "analizar", reply, RecordOptions{}); r.Saved != 1 {
		t.Fatalf("first: %+v", r)
	}
	if r, _ := RecordMemory(ad, root, "p", "analizar", reply, RecordOptions{}); r.Saved != 0 || r.Skipped != 1 {
		t.Fatalf("identical synthesis must be skipped, got %+v", r)
	}
	mem, _ := ad.GetMemory("p")
	if strings.Count(mem, "sha=") != 1 || !strings.Contains(mem, "— tarea: analizar") || !strings.Contains(mem, "_mapOrderHistory") {
		t.Fatalf("bad memory format:\n%s", mem)
	}
	if strings.Contains(mem, "YYYY-MM-DD") {
		t.Fatalf("model's placeholder heading must be replaced by Mova's:\n%s", mem)
	}
	// dos bloques en una respuesta (modo "todas"): uno por tarea, con "---" interno neutralizado
	two := longReply("```memory\n**Tarea:** analizar\n**Realizado:** A\n---\nx\n```\n```memory\n**Tarea:** agregar-columnas\n**Realizado:** B\n```")
	r, _ := RecordMemory(ad, root, "p", TaskAll, two, RecordOptions{})
	if r.Saved != 2 || strings.Join(r.Tasks, ",") != "analizar,agregar-columnas" {
		t.Fatalf("two blocks → two tagged entries, got %+v", r)
	}
	mem, _ = ad.GetMemory("p")
	if got := len(strings.Split(strings.TrimSpace(mem), memorySep)); got != 3 {
		t.Fatalf("a '---' inside a block must not split the entry: %d entries\n%s", got, mem)
	}
}

func TestRecordMemoryFallbackAndShort(t *testing.T) {
	root := t.TempDir()
	writeProj(t, root, `,"memory":true`)
	ad := NewFileAdapter(root)
	if r, _ := RecordMemory(ad, root, "p", "analizar", "ok listo", RecordOptions{}); !r.Short || r.Saved != 0 {
		t.Fatalf("short reply must not be recorded: %+v", r)
	}
	plain := "# Hallazgos\nTexto de relleno " + strings.Repeat("x", 250) + "\n- `planner.js::saveOrder` descarta FIELD_C al guardar\nOtra frase sin datos técnicos.\n"
	if r, _ := RecordMemory(ad, root, "p", "analizar", plain, RecordOptions{}); r.Saved != 1 {
		t.Fatalf("no block → automatic digest: %+v", r)
	}
	mem, _ := ad.GetMemory("p")
	if !strings.Contains(mem, "Resumen automático") || !strings.Contains(mem, "saveOrder") || strings.Contains(mem, "Otra frase sin datos") {
		t.Fatalf("digest should keep technical lines only:\n%s", mem)
	}
}

// El requisito central: una tarea trabaja DESPUÉS, en otra sesión, con lo que hizo la anterior.
func TestMemoryFlowsToOtherTasksAcrossSessions(t *testing.T) {
	root := t.TempDir()
	writeProj(t, root, `,"memory":true`)
	if r, _ := RecordMemory(NewFileAdapter(root), root, "p", "analizar", longReply(memBlock), RecordOptions{}); r.Saved != 1 {
		t.Fatal("not saved")
	}
	for _, task := range []string{"agregar-columnas", "analizar", TaskAll} { // adaptador nuevo = proceso nuevo
		s, err := BuildContextSections(NewFileAdapter(root), root, "p", task)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(s.Memory, "## MEMORY") || !strings.Contains(s.Memory, "FIELD_B — se pierde") {
			t.Fatalf("task %q does not see the saved synthesis:\n%s", task, s.Memory)
		}
	}
}

func TestMemoryCustomPathHonoredByReadWriteArchive(t *testing.T) {
	root := t.TempDir()
	ext := filepath.Join(t.TempDir(), "red", "memoria")
	writeProj(t, root, fmt.Sprintf(`,"memory":%q`, ext+string(filepath.Separator)))
	ad := NewFileAdapter(root)
	_ = ad.AppendMemory("p", "## 2020-01-05 — vieja\ncontenido viejo")
	_ = ad.AppendMemory("p", "## "+"2999-01-01"+" — nueva\ncontenido nuevo")
	if _, err := os.Stat(filepath.Join(ext, "memory.md")); err != nil {
		t.Fatalf("memory.md must live in the configured folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "projects", "p", "memory.md")); err == nil {
		t.Fatal("nothing may be written to the default location")
	}
	if err := ad.ArchiveMemory("p", 30); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ext, "memory-archive", "2020-01.md")); err != nil {
		t.Fatalf("archive must sit next to the configured memory.md: %v", err)
	}
	if m, _ := ad.GetMemory("p"); strings.Contains(m, "contenido viejo") || !strings.Contains(m, "contenido nuevo") {
		t.Fatalf("active memory after archive:\n%s", m)
	}
}

func TestAppendMemoryConcurrentNoLostEntries(t *testing.T) {
	root := t.TempDir()
	writeProj(t, root, `,"memory":true`)
	ad := NewFileAdapter(root)
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := ad.AppendMemory("p", fmt.Sprintf("## 2026-10-01 — e%d\nentrada-%d", i, i)); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	mem, _ := ad.GetMemory("p")
	for i := 0; i < 25; i++ {
		if !strings.Contains(mem, fmt.Sprintf("entrada-%d\n", i)) && !strings.HasSuffix(strings.TrimSpace(mem), fmt.Sprintf("entrada-%d", i)) && !strings.Contains(mem, fmt.Sprintf("entrada-%d ", i)) {
			t.Fatalf("lost entry %d", i)
		}
	}
	if _, err := os.Stat(MemoryPath(root, "p") + ".lock"); err == nil {
		t.Fatal("lock file must be removed")
	}
}

func entryFor(task string, day int, body string) string {
	return fmt.Sprintf("## 2026-10-%02d 10:00 — tarea: %s\n<!-- mova:entry task=%s sha=%012d -->\n%s", day, task, task, day, body)
}

func TestFormatMemoryForContext(t *testing.T) {
	small := entryFor("analizar", 1, "corto")
	if got := FormatMemoryForContext(small, 0); got != small {
		t.Fatalf("under the cap the memory must be injected verbatim")
	}
	// 1 análisis antiguo + 40 entradas más nuevas de "chat": el análisis NO puede perderse.
	parts := []string{}
	for d := 40; d >= 2; d-- {
		parts = append(parts, entryFor("chat", d, strings.Repeat("ruido ", 100)))
	}
	parts = append(parts, entryFor("analizar", 1, "HALLAZGO-CRITICO: FIELD_B se pierde en _processOrder"))
	mem := strings.Join(parts, memorySep)
	out := FormatMemoryForContext(mem, 4000)
	if !strings.Contains(out, "HALLAZGO-CRITICO") {
		t.Fatalf("latest entry of every task must be pinned")
	}
	if len(out) > 4000+400 || !strings.Contains(out, "abreviadas u omitidas") {
		t.Fatalf("cap not applied (len=%d)", len(out))
	}
	if !strings.Contains(out, "tarea: chat") {
		t.Fatalf("newest entries still expected")
	}
}

// Package evidence writes one immutable record per Mova execution:
//
//	projects/<project>/runs/<run_id>/
//	  context.txt   exact bytes Mova released (or would have released)
//	  manifest.json what Mova decided, under which rules, from which code
//	  events.jsonl  append-only: later turns Mova itself observed
//	                (provider calls, tool results in its own loops,
//	                governed reads, hook decisions)
//
// context.txt and manifest.json are created with O_EXCL and never
// rewritten: a second write to the same run fails. This replaces the old
// egress_sanitized.md "block store", which merged and overwrote blocks
// from different runs and tasks and could not answer "what did run N
// release?".
//
// Every attribute says where it came from (Attr.Source): "declared"
// (written in project.json by a person), "observed" (Mova saw it happen)
// or "not_observable" (outside Mova's perimeter). Mova never records as
// observed something it did not see.
package evidence

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SchemaVersion of manifest.json.
const SchemaVersion = 1

// Attr is a value plus its provenance.
type Attr struct {
	Value  string `json:"value"`
	Source string `json:"source"` // declared:<where> | observed:<how> | not_observable:<why>
}

// RepoState is the code the context was built from.
type RepoState struct {
	Path   string `json:"path"`
	Commit string `json:"commit,omitempty"`
	Dirty  *bool  `json:"dirty,omitempty"`
	Note   string `json:"note,omitempty"`
}

// FileHash identifies a configuration file by content.
type FileHash struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ContextState describes the released bytes (context.txt).
type ContextState struct {
	SHA256          string `json:"sha256"`
	Bytes           int    `json:"bytes"`
	TokensEstimated int    `json:"tokens_estimated"`
	Tokenizer       string `json:"tokenizer"`
}

// Decision is the gate outcome Mova applied before anything left it.
type Decision struct {
	// Outcome: "released" (every gate passed; see Door for where the
	// context went), "blocked" (a gate rejected it; context.txt is the
	// governed text that was NOT released, for review), "dry_run".
	Outcome string `json:"outcome"`
	Gate    string `json:"gate,omitempty"` // which gate blocked: budget, circuit_breaker, dependency_closure, ...
	Reason  string `json:"reason,omitempty"`
}

// Manifest is manifest.json.
type Manifest struct {
	Schema    int    `json:"schema"`
	RunID     string `json:"run_id"`
	CreatedAt string `json:"created_at"`
	// Door: where the context goes if released: "cli:run" (stdout),
	// "mcp:get_full_context" (returned to the MCP host), "chat:provider"
	// (Mova calls the model), "mcp:session" (tool reads in an MCP/HTTP
	// server session), "hook:session" (hook decisions).
	Door      string          `json:"door"`
	Project   string          `json:"project"`
	Task      string          `json:"task"`
	Repo      RepoState       `json:"repo"`
	Config    []FileHash      `json:"config,omitempty"`
	Policy    string          `json:"policy_source,omitempty"`
	Spec      json.RawMessage `json:"spec,omitempty"`
	Selection []string        `json:"selection,omitempty"`
	Closure   json.RawMessage `json:"dependency_closure,omitempty"`
	Govern    json.RawMessage `json:"governance,omitempty"`
	Context   ContextState    `json:"context"`
	Agent     Attr            `json:"agent"`
	Model     Attr            `json:"model"`
	Author    Attr            `json:"policy_author"`
	Decision  Decision        `json:"decision"`
	// Perimeter is a fixed, honest statement of what this record cannot
	// contain, so a reader never mistakes it for a provider-side log.
	Perimeter string `json:"perimeter"`
}

// PerimeterNote is written into every manifest.
const PerimeterNote = "Registra solo lo que pasó por Mova. No incluye lo que un agente/IDE lea o envíe por su cuenta (sus propias tools, @-referencias, otros servidores MCP) salvo los eventos que Mova observó vía sus tools o hooks (ver events.jsonl)."

// Run is an open run directory.
type Run struct {
	ID  string
	Dir string
	mu  sync.Mutex
}

// NewID returns a sortable, unique run id: UTC timestamp + 6 random hex.
func NewID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return time.Now().UTC().Format("20060102T150405.000Z") + "-" + hex.EncodeToString(b)
}

// SHA256 of a byte slice, hex.
func SHA256(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// HashFile hashes a file; "" if unreadable.
func HashFile(path string) FileHash {
	b, err := os.ReadFile(path)
	if err != nil {
		return FileHash{Path: path}
	}
	return FileHash{Path: path, SHA256: SHA256(b)}
}

// GitState reads HEAD and dirtiness of repoDir with the git CLI. Absent
// git or a non-repo directory is recorded as a note, not invented.
func GitState(repoDir string) RepoState {
	st := RepoState{Path: repoDir}
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output()
	if err != nil {
		st.Note = "commit no disponible (git ausente o el repo no es un repositorio git)"
		return st
	}
	st.Commit = strings.TrimSpace(string(out))
	if status, err := exec.Command("git", "-C", repoDir, "status", "--porcelain", "--", ".").Output(); err == nil {
		dirty := len(strings.TrimSpace(string(status))) > 0
		st.Dirty = &dirty
	}
	return st
}

// ErrRunExists is returned when a run directory or file already exists.
var ErrRunExists = errors.New("evidence: el run ya existe (la evidencia es de escritura única)")

// Start creates runsDir/<m.RunID>/ with context.txt and manifest.json,
// both write-once. m.Context.SHA256/Bytes are computed from context.
func Start(runsDir string, m Manifest, context []byte) (*Run, error) {
	if m.RunID == "" {
		m.RunID = NewID()
	}
	m.Schema = SchemaVersion
	if m.CreatedAt == "" {
		m.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	m.Context.SHA256 = SHA256(context)
	m.Context.Bytes = len(context)
	if m.Perimeter == "" {
		m.Perimeter = PerimeterNote
	}
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return nil, err
	}
	dir := filepath.Join(runsDir, m.RunID)
	if err := os.Mkdir(dir, 0o755); err != nil {
		if os.IsExist(err) {
			return nil, ErrRunExists
		}
		return nil, err
	}
	if err := writeOnce(filepath.Join(dir, "context.txt"), context); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := writeOnce(filepath.Join(dir, "manifest.json"), append(data, '\n')); err != nil {
		return nil, err
	}
	return &Run{ID: m.RunID, Dir: dir}, nil
}

func writeOnce(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return ErrRunExists
		}
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Event appends one JSON line to events.jsonl (append-only). Kind names
// what Mova observed: "provider_call", "dry_run_block", "tool_result",
// "read", "hook_check_read", "hook_sanitize_output", "error".
func (r *Run) Event(kind string, fields map[string]any) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rec := map[string]any{"time": time.Now().UTC().Format(time.RFC3339Nano), "kind": kind}
	for k, v := range fields {
		rec[k] = v
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(r.Dir, "events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// Raw marshals v for the json.RawMessage fields of Manifest.
func Raw(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(fmt.Sprintf("%q", err.Error()))
	}
	return b
}

// sessions keeps one long-lived run per (runsDir, door) for doors that
// have no single "context release" (an MCP/HTTP server process serving
// read_file, or hook decisions): their events go to one session run.
var (
	sessionsMu sync.Mutex
	sessions   = map[string]*Run{}
)

// Session returns (creating on first use) the session run for runsDir +
// door. Its context.txt is empty: these runs only carry events.
func Session(runsDir string, m Manifest) (*Run, error) {
	key := runsDir + "|" + m.Door
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if r, ok := sessions[key]; ok {
		return r, nil
	}
	m.Decision = Decision{Outcome: "session", Reason: "run de sesión: solo eventos (lecturas/decisiones observadas por Mova)"}
	r, err := Start(runsDir, m, nil)
	if err != nil {
		return nil, err
	}
	sessions[key] = r
	return r, nil
}

// Runs lists run directories under runsDir, oldest first (ids sort by time).
func Runs(runsDir string) ([]string, error) {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(runsDir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

// ReadManifest loads runDir/manifest.json.
func ReadManifest(runDir string) (Manifest, error) {
	var m Manifest
	b, err := os.ReadFile(filepath.Join(runDir, "manifest.json"))
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(b, &m)
	return m, err
}

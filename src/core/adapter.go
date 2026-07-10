// adapter.go — storage abstraction.
// The engine never knows if data comes from files or a database.
// To add a new storage backend: implement Adapter. Backend SELECTION
// (which Adapter to instantiate for a given project) is an application-level
// decision, not a core concern — see cli/adapter_select.go, which is the
// only place that imports both core (this package) and adapters (db).
package core

// Adapter is the single storage contract.
// FileAdapter (this package) and adapters.DBAdapter both implement this.
type Adapter interface {
	GetKnowledge(kind, domain, lang, name string) (string, error)
	GetProject(name string) (*Project, error)
	ListProjects() ([]ProjectSummary, error)
	GetMemory(project string) (string, error)
	GetMemoryAll(project string) (string, error)
	AppendMemory(project, entry string) error
	ArchiveMemory(project string, keepDays int) error
	DeleteMemory(project string, req MemoryDeleteRequest) (int, error) // returns count deleted
	Search(query, domain string) ([]SearchResult, error)
}

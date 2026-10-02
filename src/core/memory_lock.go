// memory_lock.go — escritura segura de memory.md: chat, MCP y HTTP pueden
// escribir a la vez (mismo proceso o procesos distintos). Antes
// AppendMemory leía y reescribía sin bloqueo y dos escrituras simultáneas
// perdían una entrada. Ahora: mutex por archivo + archivo ".lock" entre
// procesos + escritura atómica (temporal + rename).
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var memoryMutexes sync.Map

const (
	memoryLockWait  = 10 * time.Second
	memoryLockStale = 30 * time.Second
)

func withMemoryLock(path string, fn func() error) error {
	mu, _ := memoryMutexes.LoadOrStore(path, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	lock := path + ".lock"
	deadline := time.Now().Add(memoryLockWait)
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			f.Close()
			defer os.Remove(lock)
			break
		}
		if !os.IsExist(err) {
			break // sin permiso para crear el .lock: sigue con el mutex del proceso
		}
		if st, serr := os.Stat(lock); serr == nil && time.Since(st.ModTime()) > memoryLockStale {
			_ = os.Remove(lock) // bloqueo huérfano de un proceso caído
			continue
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("memory: %s está bloqueado por otro proceso", path)
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fn()
}

// writeFileAtomic escribe en un temporal del mismo directorio y renombra.
func writeFileAtomic(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp-%d", path, os.Getpid())
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path) // Windows: rename no pisa un archivo existente
		if err = os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return nil
}

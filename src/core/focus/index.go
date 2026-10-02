// index.go — mapa de búsqueda rápida (nombre de archivo → rutas) para
// `focus`/`exclude`. Se construye con UNA sola caminata del repo por
// ejecución y se comparte entre todos los targets y resolvers vía
// Context.Index, en vez de recorrer el disco una vez por cada target
// que solo trae un nombre ("Gantt.js") o una ruta parcial.
package focus

import (
	"strings"
	"sync"
)

// FileIndex indexa los archivos del repo por nombre base en minúsculas.
// El valor cero es válido y se construye de forma perezosa.
type FileIndex struct {
	once   sync.Once
	byName map[string][]string
}

// ByName devuelve el índice, construyéndolo la primera vez con build.
// build recibe add(path, base) y lo llama por cada archivo, en orden
// determinista (lo garantiza quien recorre el filesystem).
func (i *FileIndex) ByName(build func(add func(path, base string))) map[string][]string {
	i.once.Do(func() {
		i.byName = map[string][]string{}
		build(func(path, base string) {
			k := strings.ToLower(base)
			i.byName[k] = append(i.byName[k], path)
		})
	})
	return i.byName
}

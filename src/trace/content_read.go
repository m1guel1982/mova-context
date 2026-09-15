// content_read.go — per-file content/tokenization helpers used by
// AnalyzeRemote's walk (see analyzer.go): image token estimation and
// safe text/binary reading. Split out of analyzer.go purely to keep
// every file in this package under the 300-line limit.
package trace

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"mova.local/documents"
)

func estimateImageTokens(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 765
	}
	defer file.Close()

	// Intenta leer las dimensiones nativas
	cfg, _, err := image.DecodeConfig(file)
	if err == nil && cfg.Width > 0 && cfg.Height > 0 {
		width := float64(cfg.Width)
		height := float64(cfg.Height)

		// 1. Escalar si supera 2048px en lado mayor
		maxDim := math.Max(width, height)
		if maxDim > 2048 {
			scale := 2048.0 / maxDim
			width *= scale
			height *= scale
		}

		// 2. Escalar lado menor a 768px
		minDim := math.Min(width, height)
		if minDim > 768 {
			scale := 768.0 / minDim
			width *= scale
			height *= scale
		}

		// 3. Bloques de 512x512
		tilesX := math.Ceil(width / 512.0)
		tilesY := math.Ceil(height / 512.0)

		return 85 + int(tilesX*tilesY)*170
	}

	// Fallback por peso de archivo si image.DecodeConfig no reconoce la cabecera
	info, err := file.Stat()
	if err != nil {
		return 765
	}

	size := info.Size()
	switch {
	case size < 100*1024: // Menor a 100 KB (imágenes pequeñas / iconos)
		return 255
	case size < 500*1024: // Entre 100 KB y 500 KB (resolución estándar)
		return 765
	case size < 2*1024*1024: // Entre 500 KB y 2 MB (fotos HD)
		return 1105
	default: // Mayor a 2 MB (resolución ultra alta)
		return 1445
	}
}

// getFileContentForTokenization lee y normaliza el contenido del archivo.
// Devuelve el texto (si aplica), o los tokens directos (si es una imagen) y el error.
func getFileContentForTokenization(path string) (string, int, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".ico", ".webp", ".bmp":
		// Para imágenes, retornamos string vacío y calculamos los tokens de visión
		return "", estimateImageTokens(path), nil

	case ".exe", ".zip", ".tar", ".gz":
		return "", 0, nil

	case ".pdf", ".docx", ".xlsx":
		text, err := documents.ReadDocumentLayer(path)
		if err != nil {
			return "", 0, nil
		}
		return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"), 0, nil

	default:
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			return "", 0, nil
		}

		// Si el archivo contiene bytes nulos (0x00), es un archivo binario y lo omitimos
		if isBinaryContent(data) {
			return "", 0, nil
		}

		str := string(data)
		str = strings.ReplaceAll(str, "\r\n", "\n")
		str = strings.ReplaceAll(str, "\r", "\n")
		return str, 0, nil
	}
}

func isBinaryContent(data []byte) bool {
	// Revisa los primeros 512 bytes en busca de byte nulo 0x00
	limit := len(data)
	if limit > 512 {
		limit = 512
	}
	for i := 0; i < limit; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

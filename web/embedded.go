package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
)

//go:embed all:static
var StaticFS embed.FS

//go:embed templates/*
var TemplateFS embed.FS

// CarregarTemplates compila todos os templates HTML embutidos no binário
func CarregarTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"formatBytes": func(b int64) string {
			if b <= 0 {
				return ""
			}
			const unit = 1024
			if b < unit {
				return fmt.Sprintf("%d B", b)
			}
			div, exp := int64(unit), 0
			for n := b / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}

	return template.New("").Funcs(funcMap).ParseFS(TemplateFS, "templates/*.html")
}

// withEdgeCache injeta cabeçalhos de cache otimizados para CDN e proxies de borda
func withEdgeCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
		next.ServeHTTP(w, r)
	})
}

// StaticFileServer devolve o handler para servir CSS, JS e imagens embutidas
func StaticFileServer() http.Handler {
	// Se a pasta física existir em disco, serve diretamente para suporte rápido a uploads e capas
	for _, localDir := range []string{"web/static", "/home/rmendes/biblioteca-betania/web/static"} {
		if fi, err := os.Stat(localDir); err == nil && fi.IsDir() {
			return withEdgeCache(http.FileServer(http.Dir(localDir)))
		}
	}

	// Fallback para os assets embutidos no binário
	sub, err := fs.Sub(StaticFS, "static")
	if err != nil {
		panic(err)
	}
	return withEdgeCache(http.FileServer(http.FS(sub)))
}

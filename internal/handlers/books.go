package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"biblioteca-betania/internal/database"
	"biblioteca-betania/internal/models"
)

type IndexPageData struct {
	Titulo       string
	Query        string
	DisciplinaID int64
	NivelID      int64
	Classe       int
	TipoMaterial string
	Pagina       int
	TotalPaginas int
	TotalLivros  int64
	Livros       []models.Livro
	Disciplinas  []models.Disciplina
	Niveis       []models.NivelEducacional
	Stats        *models.EstatisticasAcervo
}

// HandleIndex renderiza a página inicial com catálogo, filtros e estatísticas
func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	discID, _ := strconv.ParseInt(r.URL.Query().Get("disciplina"), 10, 64)
	nivID, _ := strconv.ParseInt(r.URL.Query().Get("nivel"), 10, 64)
	classe, _ := strconv.Atoi(r.URL.Query().Get("classe"))
	tipoMaterial := strings.TrimSpace(r.URL.Query().Get("tipo"))
	pag, _ := strconv.Atoi(r.URL.Query().Get("p"))
	if pag < 1 {
		pag = 1
	}

	filtros := database.FiltrosBusca{
		Query:        query,
		DisciplinaID: discID,
		NivelID:      nivID,
		Classe:       classe,
		TipoMaterial: tipoMaterial,
		Pagina:       pag,
		Limite:       24,
	}

	livros, total, err := h.db.ListarLivros(filtros)
	if err != nil {
		http.Error(w, "Erro ao carregar catálogo", http.StatusInternalServerError)
		return
	}

	disciplinas, _ := h.db.ObterDisciplinas()
	niveis, _ := h.db.ObterNiveis()
	stats, _ := h.db.ObterEstatisticas()

	totalPaginas := int((total + 23) / 24)
	if totalPaginas < 1 {
		totalPaginas = 1
	}

	data := IndexPageData{
		Titulo:       "BiblioTânia — Biblioteca Digital do Colégio Betânia",
		Query:        query,
		DisciplinaID: discID,
		NivelID:      nivID,
		Classe:       classe,
		TipoMaterial: tipoMaterial,
		Pagina:       pag,
		TotalPaginas: totalPaginas,
		TotalLivros:  total,
		Livros:       livros,
		Disciplinas:  disciplinas,
		Niveis:       niveis,
		Stats:        stats,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Erro ao renderizar template: "+err.Error(), http.StatusInternalServerError)
	}
}

// HandleLivroDetalhes exibe os metadados de uma obra
func (h *Handler) HandleLivroDetalhes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	livro, err := h.db.ObterLivroPorID(id)
	if err != nil || livro == nil {
		http.NotFound(w, r)
		return
	}

	_ = h.db.IncrementarVisualizacao(id)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(livro)
}

// HandleLerLivro redireciona para link externo ou serve o PDF enviado com proteção
func (h *Handler) HandleLerLivro(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	livro, err := h.db.ObterLivroPorID(id)
	if err != nil || livro == nil {
		http.NotFound(w, r)
		return
	}

	_ = h.db.IncrementarDownload(id)

	// Se for link externo público, redireciona 302
	if livro.LinkExterno != "" {
		http.Redirect(w, r, livro.LinkExterno, http.StatusFound)
		return
	}

	// Se for arquivo local carregado, servir com path sanitizado
	if livro.ArquivoURL != "" {
		baseName := filepath.Base(livro.ArquivoURL)
		safePath := filepath.Join(h.cfg.UploadDir, baseName)

		if _, err := os.Stat(safePath); os.IsNotExist(err) {
			http.Error(w, "Arquivo não encontrado nos servidores", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		safeTitle := strings.ReplaceAll(livro.Titulo, "\"", "'")
		safeTitle = strings.ReplaceAll(safeTitle, "\r", "")
		safeTitle = strings.ReplaceAll(safeTitle, "\n", " ")
		safeTitle = strings.TrimSpace(safeTitle)
		if safeTitle == "" {
			safeTitle = "documento"
		}
		w.Header().Set("Content-Disposition", "inline; filename=\""+safeTitle+".pdf\"")
		http.ServeFile(w, r, safePath)
		return
	}

	http.NotFound(w, r)
}

// HandleAPILivros devolve catálogo em formato JSON para autocomplete e agentes
func (h *Handler) HandleAPILivros(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	discID, _ := strconv.ParseInt(r.URL.Query().Get("disciplina"), 10, 64)
	nivID, _ := strconv.ParseInt(r.URL.Query().Get("nivel"), 10, 64)
	pag, _ := strconv.Atoi(r.URL.Query().Get("p"))

	filtros := database.FiltrosBusca{
		Query:        query,
		DisciplinaID: discID,
		NivelID:      nivID,
		Pagina:       pag,
		Limite:       24,
	}

	livros, total, err := h.db.ListarLivros(filtros)
	if err != nil {
		http.Error(w, `{"error":"falha ao consultar livros"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"total":  total,
		"pagina": pag,
		"livros": livros,
	})
}

// HandleAPIStats retorna estatísticas do acervo
func (h *Handler) HandleAPIStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.ObterEstatisticas()
	if err != nil {
		http.Error(w, `{"error":"falha ao obter estatísticas"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// HandleSobre renderiza a página institucional da BiblioTânia
func (h *Handler) HandleSobre(w http.ResponseWriter, r *http.Request) {
	stats, _ := h.db.ObterEstatisticas()
	data := map[string]any{
		"Titulo": "Sobre a BiblioTânia — Colégio Betânia",
		"Stats":  stats,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "sobre.html", data); err != nil {
		http.Error(w, "Erro ao renderizar página: "+err.Error(), http.StatusInternalServerError)
	}
}

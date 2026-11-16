package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"biblioteca-betania/internal/models"
)

type SubmeterPageData struct {
	Titulo      string
	Disciplinas []models.Disciplina
	Niveis      []models.NivelEducacional
	Mensagem    string
	TipoErro    bool
}

// HandleSubmeterPagina exibe o formulário de envio de livros para os estudantes
func (h *Handler) HandleSubmeterPagina(w http.ResponseWriter, r *http.Request) {
	disciplinas, _ := h.db.ObterDisciplinas()
	niveis, _ := h.db.ObterNiveis()

	msg := r.URL.Query().Get("msg")
	erro := r.URL.Query().Get("erro") == "1"

	data := SubmeterPageData{
		Titulo:      "Sugerir Livro — BiblioTânia (Colégio Betânia)",
		Disciplinas: disciplinas,
		Niveis:      niveis,
		Mensagem:    msg,
		TipoErro:    erro,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "submeter.html", data); err != nil {
		http.Error(w, "Erro ao renderizar formulário: "+err.Error(), http.StatusInternalServerError)
	}
}

// HandleProcessarSubmissao recebe o envio multipart/form-data do estudante
func (h *Handler) HandleProcessarSubmissao(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Limitar tamanho do request (prevenção de DoS por memória)
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxFileSize+1024*1024)
	if err := r.ParseMultipartForm(h.cfg.MaxFileSize); err != nil {
		http.Redirect(w, r, "/submeter?erro=1&msg=Arquivo+excede+o+limite+permitido+(max+50MB)", http.StatusSeeOther)
		return
	}

	nome := strings.TrimSpace(r.FormValue("estudante_nome"))
	classe := strings.TrimSpace(r.FormValue("estudante_classe"))
	titulo := strings.TrimSpace(r.FormValue("titulo"))
	autor := strings.TrimSpace(r.FormValue("autor"))
	tipoEnvio := strings.TrimSpace(r.FormValue("tipo_envio")) // "arquivo" ou "link"
	linkURL := strings.TrimSpace(r.FormValue("link_url"))
	discID, _ := strconv.ParseInt(r.FormValue("disciplina_id"), 10, 64)
	nivID, _ := strconv.ParseInt(r.FormValue("nivel_id"), 10, 64)

	if nome == "" || classe == "" || titulo == "" {
		responderSubmissaoErro(w, r, "Por favor preencha seu nome, classe e titulo do livro", http.StatusBadRequest)
		return
	}

	sub := &models.SubmissaoEstudante{
		EstudanteNome:   nome,
		EstudanteClasse: classe,
		Titulo:          titulo,
		Autor:           autor,
		DisciplinaID:    discID,
		NivelID:         nivID,
		TipoEnvio:       tipoEnvio,
		Status:          "pendente",
	}

	if tipoEnvio == "link" {
		if linkURL == "" {
			responderSubmissaoErro(w, r, "Informe um link valido para o livro", http.StatusBadRequest)
			return
		}
		u, err := url.Parse(linkURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			responderSubmissaoErro(w, r, "Link invalido. Utilize um endereco iniciado por https://", http.StatusBadRequest)
			return
		}
		sub.LinkURL = linkURL
	} else {
		// Envio de arquivo físico (PDF ou DOCX)
		file, header, err := r.FormFile("arquivo")
		if err != nil {
			responderSubmissaoErro(w, r, "Selecione um arquivo PDF ou DOCX para enviar", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Sniffing de MIME type nos primeiros 512 bytes (evitar arquivos executáveis)
		buff := make([]byte, 512)
		n, _ := file.Read(buff)
		fileType := http.DetectContentType(buff[:n])
		_, _ = file.Seek(0, io.SeekStart)

		ext := strings.ToLower(filepath.Ext(header.Filename))
		isPermitido := ext == ".pdf" || ext == ".docx" || ext == ".doc" ||
			strings.Contains(fileType, "pdf") || strings.Contains(fileType, "officedocument") ||
			strings.Contains(fileType, "zip") // docx é empacotado como zip

		if !isPermitido {
			responderSubmissaoErro(w, r, "Formato invalido. Apenas arquivos PDF e DOCX sao aceitos", http.StatusBadRequest)
			return
		}

		// Gerar nome único e seguro contra path traversal
		randomBytes := make([]byte, 16)
		_, _ = rand.Read(randomBytes)
		safeFileName := fmt.Sprintf("%x%s", randomBytes, ext)
		destPath := filepath.Join(h.cfg.UploadDir, safeFileName)

		// Criar pasta se não existir
		_ = os.MkdirAll(h.cfg.UploadDir, 0755)

		out, err := os.Create(destPath)
		if err != nil {
			responderSubmissaoErro(w, r, "Falha interna ao armazenar arquivo", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			responderSubmissaoErro(w, r, "Falha ao escrever arquivo no disco", http.StatusInternalServerError)
			return
		}

		sub.ArquivoPath = safeFileName
		sub.ArquivoNomeOriginal = filepath.Base(header.Filename)
	}

	id, err := h.db.InserirSubmissao(sub)
	if err != nil {
		responderSubmissaoErro(w, r, "Falha ao registrar submissao", http.StatusInternalServerError)
		return
	}

	// Notificar o Worker de curadoria para triagem imediata
	h.curator.NotificarNovaSubmissao()

	msgSucesso := fmt.Sprintf("Submissão #%d enviada com sucesso! A curadoria pedagógica avaliará a obra.", id)
	if isJSONRequest(r) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"id":       id,
			"mensagem": msgSucesso,
		})
		return
	}
	http.Redirect(w, r, "/submissoes?msg="+url.QueryEscape(msgSucesso), http.StatusSeeOther)
}

func isJSONRequest(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json") || r.URL.Query().Get("format") == "json"
}

func responderSubmissaoErro(w http.ResponseWriter, r *http.Request, msg string, statusCode int) {
	if isJSONRequest(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": msg})
		return
	}
	http.Redirect(w, r, "/submeter?erro=1&msg="+url.QueryEscape(msg), http.StatusSeeOther)
}

// HandleAPISubmissoes devolve as submissões recentes em formato JSON para o portal e agentes
func (h *Handler) HandleAPISubmissoes(w http.ResponseWriter, r *http.Request) {
	submissoes, err := h.db.ListarSubmissoesRecentes(50)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Erro ao listar submissões"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(submissoes)
}

// HandleSubmissoesStatus exibe o mural de submissões recentes e o parecer da IA
func (h *Handler) HandleSubmissoesStatus(w http.ResponseWriter, r *http.Request) {
	submissoes, err := h.db.ListarSubmissoesRecentes(50)
	if err != nil {
		http.Error(w, "Erro ao listar submissões", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Titulo":     "Mural de Curadoria Pedagógica — BiblioTânia",
		"Submissoes": submissoes,
		"Mensagem":   r.URL.Query().Get("msg"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "submissoes.html", data); err != nil {
		http.Error(w, "Erro ao renderizar mural: "+err.Error(), http.StatusInternalServerError)
	}
}

// TokenHex helper para gerar hashes aleatórios
func TokenHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

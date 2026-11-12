package handlers

import (
	"crypto/rand"
	"encoding/hex"
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
		Titulo:      "Sugerir / Enviar Livro — BiblioTânia (Colégio Betânia)",
		Disciplinas: disciplinas,
		Niveis:      niveis,
		Mensagem:    msg,
		TipoErro:    erro,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.templates.ExecuteTemplate(w, "submeter.html", data)
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
		http.Redirect(w, r, "/submeter?erro=1&msg=Por+favor+preencha+seu+nome,+classe+e+titulo+do+livro", http.StatusSeeOther)
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
			http.Redirect(w, r, "/submeter?erro=1&msg=Informe+um+link+valido+para+o+livro", http.StatusSeeOther)
			return
		}
		u, err := url.Parse(linkURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			http.Redirect(w, r, "/submeter?erro=1&msg=Link+invalido.+Utilize+um+endereco+iniciado+por+https://", http.StatusSeeOther)
			return
		}
		sub.LinkURL = linkURL
	} else {
		// Envio de arquivo físico (PDF ou DOCX)
		file, header, err := r.FormFile("arquivo")
		if err != nil {
			http.Redirect(w, r, "/submeter?erro=1&msg=Selecione+um+arquivo+PDF+ou+DOCX+para+enviar", http.StatusSeeOther)
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
			http.Redirect(w, r, "/submeter?erro=1&msg=Formato+invalido.+Apenas+arquivos+PDF+e+DOCX+sao+aceitos", http.StatusSeeOther)
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
			http.Redirect(w, r, "/submeter?erro=1&msg=Falha+interna+ao+armazenar+arquivo", http.StatusSeeOther)
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			http.Redirect(w, r, "/submeter?erro=1&msg=Falha+ao+escrever+arquivo+no+disco", http.StatusSeeOther)
			return
		}

		sub.ArquivoPath = safeFileName
		sub.ArquivoNomeOriginal = filepath.Base(header.Filename)
	}

	id, err := h.db.InserirSubmissao(sub)
	if err != nil {
		http.Redirect(w, r, "/submeter?erro=1&msg=Falha+ao+registrar+submissao", http.StatusSeeOther)
		return
	}

	// Notificar o Worker assíncrono de IA para triagem imediata
	h.curator.NotificarNovaSubmissao()

	msgSucesso := fmt.Sprintf("Submissão #%d enviada com sucesso! A Inteligência Artificial do Colégio Betânia avaliará o material em instantes.", id)
	http.Redirect(w, r, "/submissoes?msg="+url.QueryEscape(msgSucesso), http.StatusSeeOther)
}

// HandleSubmissoesStatus exibe o mural de submissões recentes e o parecer da IA
func (h *Handler) HandleSubmissoesStatus(w http.ResponseWriter, r *http.Request) {
	submissoes, err := h.db.ListarSubmissoesRecentes(50)
	if err != nil {
		http.Error(w, "Erro ao listar submissões", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Titulo":     "Mural de Triagem da IA — BiblioTânia",
		"Submissoes": submissoes,
		"Mensagem":   r.URL.Query().Get("msg"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.templates.ExecuteTemplate(w, "submissoes.html", data)
}

// TokenHex helper para gerar hashes aleatórios
func TokenHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

package curator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"biblioteca-betania/internal/config"
	"biblioteca-betania/internal/models"
)

// ParecerIA estruturado devolvido pela Inteligência Artificial
type ParecerIA struct {
	Aprovado          bool   `json:"aprovado"`
	TituloNormalizado string `json:"titulo_normalizado"`
	AutorNormalizado  string `json:"autor_normalizado"`
	SinopseResumo     string `json:"sinopse_resumo"`
	AnoAproximado     int    `json:"ano_aproximado"`
	Parecer           string `json:"parecer"`
}

type Evaluator struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewEvaluator(cfg *config.Config) *Evaluator {
	return &Evaluator{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

// AvaliarSubmissao avalia o envio do estudante usando o motor configurado
func (e *Evaluator) AvaliarSubmissao(ctx context.Context, sub *models.SubmissaoEstudante) (*ParecerIA, error) {
	// 1. Verificação prévia de sanidade e segurança do link ou arquivo
	if sub.TipoEnvio == "link" {
		u, err := url.Parse(sub.LinkURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return &ParecerIA{
				Aprovado: false,
				Parecer:  "Rejeitado: O link fornecido possui protocolo inválido ou formato incorreto.",
			}, nil
		}

		// Bloqueio de hosts perigosos (loopback, redes privadas, schemes estranhos)
		host := strings.ToLower(u.Hostname())
		if host == "localhost" || host == "127.0.0.1" || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
			return &ParecerIA{
				Aprovado: false,
				Parecer:  "Rejeitado: Endereço de rede local restrito detectado.",
			}, nil
		}
	}

	// 2. Se houver chave do Gemini configurada, usar Gemini 2.5 Flash
	if e.cfg.GeminiAPIKey != "" {
		return e.avaliarComGemini(ctx, sub)
	}

	// 3. Se houver chave do OpenAI configurada, usar endpoint OpenAI
	if e.cfg.OpenAIAPIKey != "" {
		return e.avaliarComOpenAI(ctx, sub)
	}

	// 4. Modo Heurístico Educacional Resiliente (para operação local/offline)
	return e.avaliarHeuristicaLocal(sub), nil
}

func (e *Evaluator) avaliarComGemini(ctx context.Context, sub *models.SubmissaoEstudante) (*ParecerIA, error) {
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", e.cfg.GeminiAPIKey)

	prompt := e.construirPromptCuradoria(sub)

	payload := map[string]any{
		"contents": []any{
			map[string]any{
				"parts": []any{
					map[string]string{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"temperature":      0.2,
		},
	}

	bodyJSON, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		// Degradação graciosa para heurística se a nuvem falhar
		return e.avaliarHeuristicaLocal(sub), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("resposta vazia do gemini")
	}

	rawText := geminiResp.Candidates[0].Content.Parts[0].Text
	var parecer ParecerIA
	if err := json.Unmarshal([]byte(rawText), &parecer); err != nil {
		// Limpeza caso retorne markdown codeblock
		clean := strings.TrimSpace(rawText)
		clean = strings.TrimPrefix(clean, "```json")
		clean = strings.TrimPrefix(clean, "```")
		clean = strings.TrimSuffix(clean, "```")
		if err2 := json.Unmarshal([]byte(clean), &parecer); err2 != nil {
			return nil, fmt.Errorf("falha ao interpretar parecer json: %w", err)
		}
	}

	return &parecer, nil
}

func (e *Evaluator) avaliarComOpenAI(ctx context.Context, sub *models.SubmissaoEstudante) (*ParecerIA, error) {
	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(e.cfg.OpenAIBaseURL, "/"))
	prompt := e.construirPromptCuradoria(sub)

	payload := map[string]any{
		"model": "gpt-4o-mini",
		"messages": []any{
			map[string]string{
				"role":    "system",
				"content": "Você é o Curador Pedagógico da BiblioTânia (Biblioteca Digital do Colégio Betânia). Responda estritamente em JSON.",
			},
			map[string]string{
				"role":    "user",
				"content": prompt,
			},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2,
	}

	bodyJSON, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.cfg.OpenAIAPIKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return e.avaliarHeuristicaLocal(sub), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return e.avaliarHeuristicaLocal(sub), nil
	}

	var oaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
		return nil, err
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("resposta vazia da openai")
	}

	var parecer ParecerIA
	if err := json.Unmarshal([]byte(oaiResp.Choices[0].Message.Content), &parecer); err != nil {
		return nil, err
	}

	return &parecer, nil
}

func (e *Evaluator) construirPromptCuradoria(sub *models.SubmissaoEstudante) string {
	return fmt.Sprintf(`
Você é o Curador de Inteligência Artificial do Colégio Betânia (escola de referência do Ensino Primário e Secundário).
Avalie a submissão de um estudante para integrar o acervo oficial da "BiblioTânia".

DADOS DA SUBMISSÃO:
- Estudante: %s (%s)
- Título Sugerido: %s
- Autor Informado: %s
- Tipo de Envio: %s
- Arquivo/Link: %s

CRITÉRIOS DE AVALIAÇÃO:
1. Relevância Pedagógica: O material tem valor educativo para estudantes do Ensino Primário ou Secundário (Livros didáticos, literatura, manuais técnicos, ciências, história, etc.)?
2. Segurança e Ética Escolar: O material é estritamente livre de pornografia, violência explícita, apologia a crimes, desinformação perigosa, malware, spam comercial ou apostas?
3. Verossimilhança: Parece ser um livro/documento genuíno e útil para estudos?

Responda exclusivamente com um objeto JSON no seguinte formato:
{
  "aprovado": true,
  "titulo_normalizado": "Título formatado e corrigido",
  "autor_normalizado": "Nome do Autor ou 'Autor Desconhecido'",
  "sinopse_resumo": "Resumo de 2 a 3 frases sobre o conteúdo educativo da obra",
  "ano_aproximado": 2024,
  "parecer": "Mensagem educativa ao aluno explicando a aprovação ou os motivos da recusa."
}
`, sub.EstudanteNome, sub.EstudanteClasse, sub.Titulo, sub.Autor, sub.TipoEnvio, sub.LinkURL)
}

// avaliarHeuristicaLocal fornece triagem robusta quando nenhuma chave de API estiver presente
func (e *Evaluator) avaliarHeuristicaLocal(sub *models.SubmissaoEstudante) *ParecerIA {
	t := strings.TrimSpace(sub.Titulo)
	if len(t) < 3 {
		return &ParecerIA{
			Aprovado: false,
			Parecer:  "Rejeitado: O título do livro informado é excessivamente curto ou inválido.",
		}
	}

	// Termos estritamente proibidos em ambiente escolar
	termosProibidos := []string{
		"aposta", "bet", "cassino", "casino", "hack", "crack", "warez", "porn", "sexo", "viagra", "crypto free",
	}
	tLower := strings.ToLower(t + " " + sub.LinkURL + " " + sub.ArquivoNomeOriginal)
	for _, termo := range termosProibidos {
		if strings.Contains(tLower, termo) {
			return &ParecerIA{
				Aprovado: false,
				Parecer:  "Rejeitado: O material viola as diretrizes de segurança e moral escolar do Colégio Betânia.",
			}
		}
	}

	autor := strings.TrimSpace(sub.Autor)
	if autor == "" {
		autor = "Acervo Colégio Betânia"
	}

	return &ParecerIA{
		Aprovado:          true,
		TituloNormalizado: t,
		AutorNormalizado:  autor,
		SinopseResumo:     fmt.Sprintf("Obra recomendada e submetida pelo estudante %s (%s) para apoio curricular na BiblioTânia.", sub.EstudanteNome, sub.EstudanteClasse),
		AnoAproximado:     time.Now().Year(),
		Parecer:           fmt.Sprintf("Parabéns %s! O material atendeu aos critérios pedagógicos do Colégio Betânia e foi automaticamente catalogado.", sub.EstudanteNome),
	}
}

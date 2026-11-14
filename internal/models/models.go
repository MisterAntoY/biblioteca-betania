package models

import "time"

// Disciplina representa uma matéria escolar no Colégio Betânia
type Disciplina struct {
	ID    int64  `json:"id"`
	Nome  string `json:"nome"`
	Slug  string `json:"slug"`
	Icone string `json:"icone"`
	Ordem int    `json:"ordem"`
}

// NivelEducacional representa a faixa/ciclo escolar
type NivelEducacional struct {
	ID   int64  `json:"id"`
	Nome string `json:"nome"`
	Slug string `json:"slug"`
}

// Livro representa uma obra no acervo da BiblioTânia
type Livro struct {
	ID             int64     `json:"id"`
	Titulo         string    `json:"titulo"`
	Autor          string    `json:"autor"`
	DisciplinaID   int64     `json:"disciplina_id"`
	DisciplinaNome string    `json:"disciplina_nome,omitempty"`
	NivelID        int64     `json:"nivel_id"`
	NivelNome      string    `json:"nivel_nome,omitempty"`
	Sinopse        string    `json:"sinopse"`
	AnoPublicacao  int       `json:"ano_publicacao"`
	Classe         int       `json:"classe"` // 1..12, 0 para geral
	TipoMaterial   string    `json:"tipo_material"` // manual_oficial, literatura, apoio
	CapaURL        string    `json:"capa_url,omitempty"`
	Formato        string    `json:"formato"` // pdf, docx, link
	ArquivoURL     string    `json:"arquivo_url,omitempty"`
	LinkExterno    string    `json:"link_externo,omitempty"`
	TamanhoBytes   int64     `json:"tamanho_bytes"`
	Status         string    `json:"status"` // ativo, pendente, rejeitado
	SubmetidoPor   string    `json:"submetido_por"`
	Visualizacoes  int64     `json:"visualizacoes"`
	Downloads      int64     `json:"downloads"`
	CriadoEm       time.Time `json:"criado_em"`
}

// SubmissaoEstudante representa o envio de um livro para triagem da IA
type SubmissaoEstudante struct {
	ID                  int64      `json:"id"`
	EstudanteNome       string     `json:"estudante_nome"`
	EstudanteClasse     string     `json:"estudante_classe"`
	Titulo              string     `json:"titulo"`
	Autor               string     `json:"autor"`
	DisciplinaID        int64      `json:"disciplina_id"`
	NivelID             int64      `json:"nivel_id"`
	TipoEnvio           string     `json:"tipo_envio"` // arquivo, link
	ArquivoPath         string     `json:"arquivo_path,omitempty"`
	ArquivoNomeOriginal string     `json:"arquivo_nome_original,omitempty"`
	LinkURL             string     `json:"link_url,omitempty"`
	Status              string     `json:"status"` // pendente, aprovado, rejeitado, erro
	AIParecer           string     `json:"ai_parecer"`
	AIAvaliadoEm        *time.Time `json:"ai_avaliado_em,omitempty"`
	LivroID             *int64     `json:"livro_id,omitempty"`
	CriadoEm            time.Time  `json:"criado_em"`
}

// EstatisticasAcervo métricas rápidas de escala
type EstatisticasAcervo struct {
	TotalLivros       int64            `json:"total_livros"`
	MetaGlobal        int64            `json:"meta_global"` // 100 Milhões
	TotalSubmissoes   int64            `json:"total_submissoes"`
	TotalAprovadosIA  int64            `json:"total_aprovados_ia"`
	LivrosPorMateria  map[string]int64 `json:"livros_por_materia"`
}

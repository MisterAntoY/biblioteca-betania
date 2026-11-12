package curator

import (
	"context"
	"log"
	"time"

	"biblioteca-betania/internal/database"
	"biblioteca-betania/internal/models"
)

type Worker struct {
	db        *database.DB
	evaluator *Evaluator
	notifyCh  chan struct{}
	stopCh    chan struct{}
}

func NewWorker(db *database.DB, evaluator *Evaluator) *Worker {
	return &Worker{
		db:        db,
		evaluator: evaluator,
		notifyCh:  make(chan struct{}, 100),
		stopCh:    make(chan struct{}),
	}
}

// Start inicia o loop de processamento em segundo plano
func (w *Worker) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopCh:
				return
			case <-w.notifyCh:
				w.processarFila()
			case <-ticker.C:
				w.processarFila()
			}
		}
	}()
}

// Stop finaliza a goroutine limpa
func (w *Worker) Stop() {
	close(w.stopCh)
}

// NotificarNovaSubmissao avisa o worker para processar imediatamente
func (w *Worker) NotificarNovaSubmissao() {
	select {
	case w.notifyCh <- struct{}{}:
	default:
	}
}

func (w *Worker) processarFila() {
	pendentes, err := w.db.ObterSubmissoesPendentes(5)
	if err != nil || len(pendentes) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, sub := range pendentes {
		parecer, err := w.evaluator.AvaliarSubmissao(ctx, &sub)
		if err != nil {
			log.Printf("[Curator] Erro ao avaliar submissão #%d: %v", sub.ID, err)
			_ = w.db.AtualizarResultadoAvaliacaoIA(ctx, sub.ID, "erro", "Erro temporário ao processar com a IA. Nova tentativa em breve.", nil)
			continue
		}

		if parecer.Aprovado {
			// Adicionar livro automaticamente ao acervo ativo
			novoLivro := &models.Livro{
				Titulo:        parecer.TituloNormalizado,
				Autor:         parecer.AutorNormalizado,
				DisciplinaID:  sub.DisciplinaID,
				NivelID:       sub.NivelID,
				Sinopse:       parecer.SinopseResumo,
				AnoPublicacao: parecer.AnoAproximado,
				Formato:       sub.TipoEnvio,
				ArquivoURL:    sub.ArquivoPath,
				LinkExterno:   sub.LinkURL,
				Status:        "ativo",
				SubmetidoPor:  sub.EstudanteNome + " (" + sub.EstudanteClasse + ")",
			}
			if sub.TipoEnvio == "arquivo" {
				novoLivro.Formato = "pdf"
			}

			livroID, err := w.db.InserirLivro(novoLivro)
			if err != nil {
				log.Printf("[Curator] Falha ao cadastrar livro aprovado da submissão #%d: %v", sub.ID, err)
				_ = w.db.AtualizarResultadoAvaliacaoIA(ctx, sub.ID, "erro", "Falha interna ao indexar livro aprovado.", nil)
				continue
			}

			_ = w.db.AtualizarResultadoAvaliacaoIA(ctx, sub.ID, "aprovado", parecer.Parecer, &livroID)
			log.Printf("[Curator] Submissão #%d APROVADA pela IA -> Livro #%d '%s'", sub.ID, livroID, novoLivro.Titulo)
		} else {
			_ = w.db.AtualizarResultadoAvaliacaoIA(ctx, sub.ID, "rejeitado", parecer.Parecer, nil)
			log.Printf("[Curator] Submissão #%d REJEITADA pela IA. Motivo: %s", sub.ID, parecer.Parecer)
		}
	}
}

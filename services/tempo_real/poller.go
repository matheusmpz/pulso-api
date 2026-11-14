// Package tempo_real detecta dados novos por polling e envia as mudanças
// dos cards pelo WebSocket (seção 13).
package tempo_real

import (
	"context"
	"log"
	"sort"
	"time"

	"pulso-api/models"
)

// FonteIDs é o que o poller precisa do banco (o EventoRepository satisfaz).
type FonteIDs interface {
	MaxIDs(ctx context.Context) (map[string]int64, error)
}

// tempoConsultaPoller é o prazo de cada consulta dos cursores.
const tempoConsultaPoller = 5 * time.Second

// Poller consulta o MAX(id) das tabelas monitoradas a cada Intervalo e,
// quando algum cresce, avisa quais eventos ocorreram.
type Poller struct {
	Fonte     FonteIDs
	Intervalo time.Duration
	// Notificar recebe os eventos do ciclo (sem repetição, ordenados).
	Notificar func(ctx context.Context, eventos []string)

	ultimos map[string]int64 // tabela → último id visto
}

// Iniciar guarda os cursores atuais, para ninguém receber uma rajada de
// eventos antigos ao conectar.
func (p *Poller) Iniciar(ctx context.Context) error {
	ctx, cancelar := context.WithTimeout(ctx, tempoConsultaPoller)
	defer cancelar()
	ids, err := p.Fonte.MaxIDs(ctx)
	if err != nil {
		return err
	}
	p.ultimos = ids
	return nil
}

// Rodar executa os ciclos até o contexto ser cancelado.
func (p *Poller) Rodar(ctx context.Context) {
	ticker := time.NewTicker(p.Intervalo)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("Tempo real: poller parado")
			return
		case <-ticker.C:
			if eventos := p.Ciclo(ctx); len(eventos) > 0 && p.Notificar != nil {
				p.Notificar(ctx, eventos)
			}
		}
	}
}

// Ciclo lê os cursores, atualiza os que cresceram e devolve os eventos
// correspondentes. Em erro de banco, loga e não muda nada (tenta de novo
// no próximo ciclo).
func (p *Poller) Ciclo(ctx context.Context) []string {
	if p.ultimos == nil {
		if err := p.Iniciar(ctx); err != nil {
			log.Println("Tempo real: erro ao iniciar os cursores: ", err)
		}
		return nil
	}

	consulta, cancelar := context.WithTimeout(ctx, tempoConsultaPoller)
	defer cancelar()
	ids, err := p.Fonte.MaxIDs(consulta)
	if err != nil {
		log.Println("Tempo real: erro ao ler os cursores: ", err)
		return nil
	}

	var eventos []string
	for evento, tabela := range models.TabelaPorEvento {
		if novo := ids[tabela]; novo > p.ultimos[tabela] {
			p.ultimos[tabela] = novo
			eventos = append(eventos, evento)
		}
	}
	sort.Strings(eventos)
	return eventos
}

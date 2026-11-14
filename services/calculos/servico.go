package calculos

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"pulso-api/models"
	"pulso-api/utils"
)

// Padrões do cálculo (seções 10 e 13.2).
const (
	ConcorrenciaPadrao = 4
	TimeoutPadrao      = 10 * time.Second
)

// Servico calcula cards a partir do registro, com concorrência limitada
// e timeout por card. A falha de um card não derruba os outros.
type Servico struct {
	Registro     Registro
	Concorrencia int
	Timeout      time.Duration
	// Agora devolve o horário atual (substituível nos testes).
	Agora func() time.Time
}

// NovoServico cria o serviço com os padrões (4 simultâneos, 10 s por card).
func NovoServico(reg Registro) *Servico {
	return &Servico{Registro: reg, Concorrencia: ConcorrenciaPadrao, Timeout: TimeoutPadrao}
}

// CalcularCards calcula os cards em paralelo (no máximo Concorrencia ao
// mesmo tempo) e devolve os resultados na mesma ordem de entrada.
func (s *Servico) CalcularCards(ctx context.Context, cards []models.CardMeta, f models.Filtros) []models.CardResultado {
	resultados := make([]models.CardResultado, len(cards))
	limite := s.Concorrencia
	if limite <= 0 {
		limite = ConcorrenciaPadrao
	}
	vagas := make(chan struct{}, limite)

	var wg sync.WaitGroup
	for i, card := range cards {
		wg.Add(1)
		go func(i int, card models.CardMeta) {
			defer wg.Done()
			vagas <- struct{}{}        // ocupa uma vaga
			defer func() { <-vagas }() // libera ao terminar
			resultados[i] = s.CalcularCard(ctx, card, f)
		}(i, card)
	}
	wg.Wait()
	return resultados
}

// CalcularCard calcula um card com timeout. Em qualquer falha (erro,
// timeout, pânico ou card sem calculador) devolve o envelope com
// "erro" e registra o motivo real no log.
func (s *Servico) CalcularCard(ctx context.Context, card models.CardMeta, f models.Filtros) models.CardResultado {
	falha := models.CardResultado{Codigo: card.Codigo, Titulo: card.Titulo, Tipo: card.Tipo, Erro: models.MsgErroCard}

	calc, ok := s.Registro[card.Codigo]
	if !ok {
		log.Printf("Cálculo: card %s sem calculador no registro", card.Codigo)
		return falha
	}

	timeout := s.Timeout
	if timeout <= 0 {
		timeout = TimeoutPadrao
	}
	ctx, cancelar := context.WithTimeout(ctx, timeout)
	defer cancelar()

	// Roda numa goroutine para o prazo valer mesmo se o calculador ignorar
	// o contexto. O canal tem buffer para a goroutine não ficar presa se
	// o prazo vencer antes.
	type saida struct {
		resultado models.CardResultado
		err       error
	}
	pronto := make(chan saida, 1)
	go func() {
		r, err := s.executar(ctx, calc, f)
		pronto <- saida{r, err}
	}()

	var resultado models.CardResultado
	var err error
	select {
	case out := <-pronto:
		resultado, err = out.resultado, out.err
	case <-ctx.Done():
		err = fmt.Errorf("prazo de %s esgotado: %w", timeout, ctx.Err())
	}
	if err != nil {
		log.Printf("Cálculo: card %s falhou: %v", card.Codigo, err)
		return falha
	}

	agora := s.agora().In(utils.FusoApp()).Truncate(time.Second)
	resultado.Codigo, resultado.Titulo, resultado.Tipo = card.Codigo, card.Titulo, card.Tipo
	resultado.Alerta = nil // limiares fora do MVP: alerta sempre null
	resultado.AtualizadoEm = &agora
	resultado.Erro = ""
	return resultado
}

// executar roda o calculador transformando um pânico em erro, para um
// card com bug não derrubar a requisição inteira.
func (s *Servico) executar(ctx context.Context, calc Calculador, f models.Filtros) (resultado models.CardResultado, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pânico no cálculo: %v", r)
		}
	}()
	return calc.Calcular(ctx, f)
}

func (s *Servico) agora() time.Time {
	if s.Agora != nil {
		return s.Agora()
	}
	return time.Now()
}

// Package calculos tem um Calculador por card e o registro que liga o
// codigo do card (tabela cards) à função de cálculo.
package calculos

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"pulso-api/models"
	"pulso-api/repositories"
)

// Calculador calcula um card. Codigo deve bater exatamente com
// cards.codigo. Gatilhos são os eventos (models.Evento*) que podem mudar
// o resultado do card; o poller do tempo real usa essa lista.
//
// Calcular devolve o CardResultado com Dados preenchido (no formato do
// tipo do card). Codigo, Titulo, Tipo, Alerta e AtualizadoEm são
// sobrescritos pelo Servico a partir da tabela cards.
type Calculador interface {
	Codigo() string
	Gatilhos() []string
	Calcular(ctx context.Context, f models.Filtros) (models.CardResultado, error)
}

// Registro liga o codigo do card ao seu Calculador.
type Registro map[string]Calculador

// ErrNaoImplementado é devolvido pelos cards cujo cálculo ainda não existe.
var ErrNaoImplementado = errors.New("cálculo do card ainda não implementado")

// NovoRegistro monta o registro com os 41 cards. Os ainda não
// implementados usam um calculador pendente (com os gatilhos reais), que
// devolve ErrNaoImplementado; cada etapa de cards troca os seus pelos
// cálculos de verdade.
func NovoRegistro(db *sql.DB) Registro {
	reg := Registro{}
	for codigo, gatilhos := range gatilhosPorCard {
		reg[codigo] = pendente{codigo: codigo, gatilhos: gatilhos}
	}
	// Cálculos reais, sobrescrevendo os pendentes
	projetos := repositories.NewMetricasProjetosRepository(db)
	ia := repositories.NewMetricasIARepository(db)

	reg.Registrar(NovoProgresso(projetos))       // proj-01
	reg.Registrar(NovoHoras(projetos))           // proj-02
	reg.Registrar(NovoCustoIAProjeto(ia))        // proj-03
	reg.Registrar(NovoBurnUp(projetos))          // proj-04
	reg.Registrar(NovoIssuesPorStatus(projetos)) // proj-05
	reg.Registrar(NovoBurndown(projetos))        // proj-06
	reg.Registrar(NovoHorasEtapa("proj-07", "analise", projetos))
	reg.Registrar(NovoHorasEtapa("proj-08", "desenvolvimento", projetos))
	reg.Registrar(NovoHorasEtapa("proj-09", "testes", projetos))
	reg.Registrar(NovoTempoMedioComplexidade(projetos)) // proj-10
	reg.Registrar(NovoTempoMedioTipo(projetos))         // proj-11
	reg.Registrar(NovoEstouros(projetos))               // proj-12
	reg.Registrar(NovoHistoricoSprints(projetos))       // proj-13
	reg.Registrar(NovoDistribuicaoHoras(projetos))      // proj-14
	reg.Registrar(NovoConsumoNoTempo(ia))               // proj-15

	vendas := repositories.NewMetricasVendasRepository(db)
	reg.Registrar(NovoReceitaFechada(vendas))     // ven-01
	reg.Registrar(NovoTaxaConversao(vendas))      // ven-02
	reg.Registrar(NovoTicketMedio(vendas))        // ven-03
	reg.Registrar(NovoCicloVenda(vendas))         // ven-04
	reg.Registrar(NovoFunilVendas(vendas))        // ven-05
	reg.Registrar(NovoPipeline(vendas))           // ven-06
	reg.Registrar(NovoMotivosPerda(vendas))       // ven-07
	reg.Registrar(NovoOrigemLeads(vendas))        // ven-08
	reg.Registrar(NovoReunioesAteFechar(vendas))  // ven-09
	reg.Registrar(NovoReceitaMensal(vendas))      // ven-10
	reg.Registrar(NovoStatusPropostas(vendas))    // ven-11
	reg.Registrar(NovoMargemComplexidade(vendas)) // ven-12
	reg.Registrar(NovoOrcadoReal(vendas))         // ven-13
	reg.Registrar(NovoMargemPrevistaReal(vendas)) // ven-14
	return reg
}

// Registrar adiciona (ou troca) o calculador do seu código.
func (r Registro) Registrar(c Calculador) {
	r[c.Codigo()] = c
}

// Validar compara o registro com os códigos dos cards ativos do banco.
// faltando: cards ativos sem calculador (a API não deve subir).
// sobrando: calculadores sem card ativo no banco (só aviso).
// As duas listas vêm ordenadas.
func (r Registro) Validar(codigosAtivos []string) (faltando, sobrando []string) {
	ativos := map[string]bool{}
	for _, c := range codigosAtivos {
		ativos[c] = true
		if _, ok := r[c]; !ok {
			faltando = append(faltando, c)
		}
	}
	for c := range r {
		if !ativos[c] {
			sobrando = append(sobrando, c)
		}
	}
	sort.Strings(faltando)
	sort.Strings(sobrando)
	return faltando, sobrando
}

// pendente é o calculador de um card ainda não implementado.
type pendente struct {
	codigo   string
	gatilhos []string
}

func (p pendente) Codigo() string     { return p.codigo }
func (p pendente) Gatilhos() []string { return p.gatilhos }
func (p pendente) Calcular(context.Context, models.Filtros) (models.CardResultado, error) {
	return models.CardResultado{}, ErrNaoImplementado
}

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

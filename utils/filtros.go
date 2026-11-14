package utils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"pulso-api/models"
)

// Limites da normalização.
const (
	diasPadraoPeriodo = 30  // período padrão: últimos 30 dias, contando hoje
	anosMaximoPeriodo = 5   // intervalo máximo entre de e ate
	maxItensLista     = 50  // itens por filtro de lista
	maxTamanhoTexto   = 100 // caracteres por item de texto livre
)

// ErroFiltro é um erro de validação dos filtros (vira 400 na resposta).
type ErroFiltro struct {
	Mensagem string
}

func (e *ErroFiltro) Error() string { return e.Mensagem }

func erroFiltro(formato string, args ...interface{}) error {
	return &ErroFiltro{Mensagem: fmt.Sprintf(formato, args...)}
}

// BuscarSprintFunc busca uma sprint pelo id. Deve devolver sql.ErrNoRows
// se ela não existir.
type BuscarSprintFunc func(ctx context.Context, id int64) (models.Sprint, error)

// aplicacao diz quais parâmetros valem em cada dashboard (seção 7.1).
type aplicacao struct {
	escopo, status, origem, modelo bool
}

var aplicacaoPorDashboard = map[string]aplicacao{
	models.DashboardProjetos: {escopo: true, status: true},
	models.DashboardVendas:   {origem: true},
	models.DashboardIA:       {escopo: true, status: true, modelo: true},
}

// EntradaDeQuery lê os filtros da query string. Listas aceitam valores
// separados por vírgula e/ou o parâmetro repetido. Só o "id" é validado
// aqui (tem de ser número); o resto é validado em NormalizarFiltros.
func EntradaDeQuery(q url.Values) (models.FiltrosEntrada, error) {
	var e models.FiltrosEntrada

	texto := func(nome string) *string {
		if _, ok := q[nome]; !ok {
			return nil
		}
		v := strings.TrimSpace(q.Get(nome))
		if v == "" {
			return nil
		}
		return &v
	}
	lista := func(nome string) []string {
		var itens []string
		for _, bruto := range q[nome] {
			itens = append(itens, strings.Split(bruto, ",")...)
		}
		return itens
	}

	e.Escopo = texto("escopo")
	e.De = texto("de")
	e.Ate = texto("ate")
	if id := texto("id"); id != nil {
		n, err := strconv.ParseInt(*id, 10, 64)
		if err != nil {
			return e, erroFiltro("id inválido: informe um número inteiro positivo.")
		}
		e.ID = &n
	}
	e.TipoProjeto = lista("tipo_projeto")
	e.StatusProjeto = lista("status_projeto")
	e.OrigemLead = lista("origem_lead")
	e.Modelo = lista("modelo")
	return e, nil
}

// NormalizarFiltros valida e normaliza os filtros para um dashboard:
//   - ignora (zera) os parâmetros que não se aplicam ao dashboard;
//   - aplica os padrões (escopo "projeto", últimos 30 dias até hoje);
//   - valida enums, datas, de ≤ ate, intervalo de até 5 anos e
//     escopo=sprint com id;
//   - em escopo=sprint, troca de/ate pelo início e fim da sprint;
//   - deixa as listas sem repetição e ordenadas (nil = todos).
//
// Erros de validação são *ErroFiltro (400). Outros erros vêm da busca da
// sprint (500).
func NormalizarFiltros(ctx context.Context, e models.FiltrosEntrada, dashboard string, hoje models.Data, buscarSprint BuscarSprintFunc) (models.Filtros, error) {
	ap, ok := aplicacaoPorDashboard[dashboard]
	if !ok {
		return models.Filtros{}, fmt.Errorf("dashboard desconhecido: %q", dashboard)
	}
	f := models.Filtros{Dashboard: dashboard}

	// Escopo e id
	if ap.escopo {
		f.Escopo = models.EscopoProjeto
		if e.Escopo != nil {
			switch *e.Escopo {
			case models.EscopoProjeto, models.EscopoTime, models.EscopoSprint:
				f.Escopo = *e.Escopo
			default:
				return f, erroFiltro("escopo inválido (%q): use projeto, time ou sprint.", *e.Escopo)
			}
		}
		if e.ID != nil {
			if *e.ID <= 0 {
				return f, erroFiltro("id inválido: informe um número inteiro positivo.")
			}
			id := *e.ID
			f.ID = &id
		}
		if f.Escopo == models.EscopoSprint && f.ID == nil {
			return f, erroFiltro("escopo=sprint exige o id da sprint.")
		}
	}

	// Período
	if f.Escopo == models.EscopoSprint {
		// O período passa a ser o da sprint, qualquer que seja o enviado
		sprint, err := buscarSprint(ctx, *f.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return f, erroFiltro("Sprint %d não encontrada.", *f.ID)
		}
		if err != nil {
			return f, err
		}
		f.De, f.Ate = sprint.Inicio, sprint.Fim
	} else {
		de, ate, err := normalizarPeriodo(e.De, e.Ate, hoje)
		if err != nil {
			return f, err
		}
		f.De, f.Ate = de, ate
	}

	// Listas
	var err error
	if f.TipoProjeto, err = normalizarLista("tipo_projeto", e.TipoProjeto, models.TiposProjeto); err != nil {
		return f, err
	}
	if ap.status {
		if f.StatusProjeto, err = normalizarLista("status_projeto", e.StatusProjeto, models.StatusProjeto); err != nil {
			return f, err
		}
	}
	if ap.origem {
		if f.OrigemLead, err = normalizarLista("origem_lead", e.OrigemLead, nil); err != nil {
			return f, err
		}
	}
	if ap.modelo {
		if f.Modelo, err = normalizarLista("modelo", e.Modelo, nil); err != nil {
			return f, err
		}
	}
	return f, nil
}

// normalizarPeriodo aplica os padrões e valida de/ate.
// Sem nada: últimos 30 dias até hoje. Só de: até hoje. Só ate: os 30
// dias terminados em ate.
func normalizarPeriodo(deTxt, ateTxt *string, hoje models.Data) (models.Data, models.Data, error) {
	ate := hoje
	if ateTxt != nil {
		d, err := ParseData(*ateTxt)
		if err != nil {
			return ate, ate, erroFiltro("data inválida em 'ate' (%q): use o formato AAAA-MM-DD.", *ateTxt)
		}
		ate = d
	}

	de := SomarDias(ate, -(diasPadraoPeriodo - 1))
	if deTxt != nil {
		d, err := ParseData(*deTxt)
		if err != nil {
			return de, ate, erroFiltro("data inválida em 'de' (%q): use o formato AAAA-MM-DD.", *deTxt)
		}
		de = d
	}

	if de.After(ate.Time) {
		return de, ate, erroFiltro("'de' (%s) não pode ser maior que 'ate' (%s).", de, ate)
	}
	if ate.After(de.AddDate(anosMaximoPeriodo, 0, 0)) {
		return de, ate, erroFiltro("O período não pode passar de %d anos.", anosMaximoPeriodo)
	}
	return de, ate, nil
}

// normalizarLista tira espaços e itens vazios, remove repetições, ordena e
// valida contra os valores permitidos (permitidos nil = texto livre).
// Lista vazia vira nil (todos).
func normalizarLista(nome string, itens []string, permitidos []models.OpcaoEnum) ([]string, error) {
	vistos := map[string]bool{}
	var saida []string
	for _, bruto := range itens {
		v := strings.TrimSpace(bruto)
		if v == "" || vistos[v] {
			continue
		}
		if len([]rune(v)) > maxTamanhoTexto {
			return nil, erroFiltro("%s: cada valor pode ter até %d caracteres.", nome, maxTamanhoTexto)
		}
		if permitidos != nil && !contemValor(permitidos, v) {
			return nil, erroFiltro("%s inválido (%q). Valores aceitos: %s.", nome, v, strings.Join(models.ValoresEnum(permitidos), ", "))
		}
		vistos[v] = true
		saida = append(saida, v)
	}
	if len(saida) > maxItensLista {
		return nil, erroFiltro("%s: informe no máximo %d valores.", nome, maxItensLista)
	}
	sort.Strings(saida)
	return saida, nil
}

func contemValor(opcoes []models.OpcaoEnum, valor string) bool {
	for _, o := range opcoes {
		if o.Valor == valor {
			return true
		}
	}
	return false
}

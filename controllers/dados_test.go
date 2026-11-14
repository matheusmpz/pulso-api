package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"pulso-api/middlewares"
	"pulso-api/models"
	"pulso-api/utils"
)

// Métodos que faltavam no repoCardFalso (definido em dashboards_test.go).

func (r *repoCardFalso) ListarCodigosAtivos(context.Context) ([]string, error) {
	return []string{"proj-01", "proj-02", "ven-01"}, r.err
}

// cardsAcessoFalsos: codigo → card. "ven-01" só a diretoria (1) vê;
// "proj-99" existe mas está inativo.
var cardsAcessoFalsos = map[string]models.CardAcesso{
	"proj-01": {CardMeta: models.CardMeta{Codigo: "proj-01", Titulo: "Progresso", Tipo: "numero"}, DashboardID: 1, DashboardChave: "projetos", Ativo: true},
	"ven-01":  {CardMeta: models.CardMeta{Codigo: "ven-01", Titulo: "Receita fechada", Tipo: "numero"}, DashboardID: 2, DashboardChave: "vendas", Ativo: true},
	"proj-99": {CardMeta: models.CardMeta{Codigo: "proj-99", Titulo: "Inativo", Tipo: "numero"}, DashboardID: 1, DashboardChave: "projetos", Ativo: false},
}

func (r *repoCardFalso) BuscarAcesso(_ context.Context, codigo string, perfilID int64) (*models.CardAcesso, error) {
	if r.err != nil {
		return nil, r.err
	}
	c, ok := cardsAcessoFalsos[codigo]
	if !ok {
		return nil, sql.ErrNoRows
	}
	c.Permitido = perfilID == 1 || c.DashboardChave == "projetos"
	return &c, nil
}

// repoSprintFalso devolve a sprint 4 (03/11 a 14/11) e nenhuma outra.
type repoSprintFalso struct{ repoFiltroFalso }

func (r *repoSprintFalso) BuscarSprint(_ context.Context, id int64) (models.Sprint, error) {
	if id != 4 {
		return models.Sprint{}, sql.ErrNoRows
	}
	return models.Sprint{ID: 4, Inicio: utils.DataDoBanco(dia("2026-11-03").Time), Fim: utils.DataDoBanco(dia("2026-11-14").Time)}, nil
}

// calculadoraFalsa registra o que recebeu e devolve um resultado por card;
// o card "proj-02" falha, como faria um card com erro.
type calculadoraFalsa struct {
	filtros models.Filtros
	cards   []string
}

func (c *calculadoraFalsa) CalcularCard(_ context.Context, card models.CardMeta, f models.Filtros) models.CardResultado {
	c.filtros = f
	c.cards = append(c.cards, card.Codigo)
	if card.Codigo == "proj-02" {
		return models.CardResultado{Codigo: card.Codigo, Titulo: card.Titulo, Tipo: card.Tipo, Erro: models.MsgErroCard}
	}
	return models.CardResultado{Codigo: card.Codigo, Titulo: card.Titulo, Tipo: card.Tipo, Dados: map[string]int{"valor": 1}}
}

func (c *calculadoraFalsa) CalcularCards(ctx context.Context, cards []models.CardMeta, f models.Filtros) []models.CardResultado {
	r := make([]models.CardResultado, len(cards))
	for i, card := range cards {
		r[i] = c.CalcularCard(ctx, card, f)
	}
	return r
}

func novoRouterDados(calc *calculadoraFalsa, cardErr error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	filtroRepo := &repoSprintFalso{}
	dash := &DashboardController{Repo: &repoDashboardFalso{}, CardRepo: &repoCardFalso{err: cardErr}, FiltroRepo: filtroRepo, Calculos: calc}
	cards := &CardController{Repo: &repoCardFalso{err: cardErr}, FiltroRepo: filtroRepo, Calculos: calc}
	r := gin.New()
	auth := middlewares.AuthMiddleware(cfgJWTTeste)
	r.GET("/dashboards/:chave/dados", auth, dash.Dados)
	r.GET("/cards/:codigo/dados", auth, cards.Dados)
	return r
}

func TestDadosDashboard(t *testing.T) {
	_ = utils.ConfigurarFuso("America/Sao_Paulo")
	diretoria := tokenDoPerfil(t, 1, "diretoria")
	lider := tokenDoPerfil(t, 2, "lider_tecnologia")

	casos := []struct {
		nome     string
		caminho  string
		token    string
		status   int
		mensagem string
	}{
		{"diretoria em projetos", "/dashboards/projetos/dados", diretoria, http.StatusOK, ""},
		{"líder em vendas", "/dashboards/vendas/dados", lider, http.StatusForbidden, utils.MsgSemAcesso},
		{"dashboard inexistente", "/dashboards/xyz/dados", diretoria, http.StatusNotFound, msgDashboardNaoEncontrado},
		{"filtro inválido", "/dashboards/projetos/dados?escopo=cliente", diretoria, http.StatusBadRequest, "escopo inválido"},
		{"data inválida", "/dashboards/projetos/dados?de=ontem", diretoria, http.StatusBadRequest, "data inválida"},
		{"sprint sem id", "/dashboards/projetos/dados?escopo=sprint", diretoria, http.StatusBadRequest, "exige o id"},
		{"sprint inexistente", "/dashboards/projetos/dados?escopo=sprint&id=5", diretoria, http.StatusBadRequest, "não encontrada"},
		{"sem token", "/dashboards/projetos/dados", "", http.StatusUnauthorized, ""},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			calc := &calculadoraFalsa{}
			status, resp := executar(novoRouterDados(calc, nil), http.MethodGet, c.caminho, "", c.token)
			if status != c.status {
				t.Fatalf("status esperado %d, obtido %d (%s)", c.status, status, resp.Erro)
			}
			if c.status != http.StatusOK {
				if c.mensagem != "" && !strings.Contains(resp.Erro, c.mensagem) {
					t.Errorf("mensagem %q não contém %q", resp.Erro, c.mensagem)
				}
				if len(calc.cards) > 0 {
					t.Error("nada deveria ser calculado quando a requisição é recusada")
				}
				return
			}

			var dados models.DadosDashboard
			if err := json.Unmarshal(resp.Dados, &dados); err != nil {
				t.Fatalf("dados inválidos: %v", err)
			}
			if dados.Dashboard.Chave != "projetos" || len(dados.Cards) != 2 {
				t.Fatalf("resposta inesperada: %+v", dados)
			}
			// Um card com erro não derruba o endpoint
			if dados.Cards[0].Erro != "" || dados.Cards[1].Erro != models.MsgErroCard {
				t.Errorf("esperado proj-01 ok e proj-02 com erro: %+v", dados.Cards)
			}
		})
	}
}

func TestDadosDashboardFiltrosAplicados(t *testing.T) {
	_ = utils.ConfigurarFuso("America/Sao_Paulo")
	calc := &calculadoraFalsa{}
	caminho := "/dashboards/projetos/dados?escopo=sprint&id=4&de=2020-01-01&tipo_projeto=web,api,web&origem_lead=site"
	status, resp := executar(novoRouterDados(calc, nil), http.MethodGet, caminho, "", tokenDoPerfil(t, 1, "diretoria"))
	if status != http.StatusOK {
		t.Fatalf("status %d (%s)", status, resp.Erro)
	}

	var bruto struct {
		Filtros map[string]interface{} `json:"filtros_aplicados"`
	}
	_ = json.Unmarshal(resp.Dados, &bruto)
	f := bruto.Filtros
	if f["escopo"] != "sprint" || f["id"] != 4.0 || f["de"] != "2026-11-03" || f["ate"] != "2026-11-14" {
		t.Errorf("escopo/período inesperados: %v", f)
	}
	if tipos, _ := f["tipo_projeto"].([]interface{}); len(tipos) != 2 || tipos[0] != "api" || tipos[1] != "web" {
		t.Errorf("tipo_projeto deveria vir normalizado [api web]: %v", f["tipo_projeto"])
	}
	if f["origem_lead"] != nil {
		t.Errorf("origem_lead não vale em projetos e deveria ser ignorado: %v", f["origem_lead"])
	}
	if calc.filtros.Escopo != "sprint" || calc.filtros.De.String() != "2026-11-03" {
		t.Errorf("o cálculo deveria receber os filtros normalizados: %+v", calc.filtros)
	}
}

func TestDadosCard(t *testing.T) {
	_ = utils.ConfigurarFuso("America/Sao_Paulo")
	diretoria := tokenDoPerfil(t, 1, "diretoria")
	lider := tokenDoPerfil(t, 2, "lider_tecnologia")

	casos := []struct {
		nome    string
		caminho string
		token   string
		repoErr error
		status  int
	}{
		{"diretoria em ven-01", "/cards/ven-01/dados", diretoria, nil, http.StatusOK},
		{"líder em proj-01", "/cards/proj-01/dados", lider, nil, http.StatusOK},
		{"líder em ven-01", "/cards/ven-01/dados", lider, nil, http.StatusForbidden},
		{"card inativo", "/cards/proj-99/dados", diretoria, nil, http.StatusNotFound},
		{"card inexistente", "/cards/ia-13/dados", diretoria, nil, http.StatusNotFound},
		{"filtro inválido", "/cards/proj-01/dados?tipo_projeto=desktop", diretoria, nil, http.StatusBadRequest},
		{"vendas ignora escopo inválido", "/cards/ven-01/dados?escopo=xyz", diretoria, nil, http.StatusOK},
		{"erro no banco", "/cards/proj-01/dados", diretoria, sql.ErrConnDone, http.StatusInternalServerError},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			calc := &calculadoraFalsa{}
			status, resp := executar(novoRouterDados(calc, c.repoErr), http.MethodGet, c.caminho, "", c.token)
			if status != c.status {
				t.Fatalf("status esperado %d, obtido %d (%s)", c.status, status, resp.Erro)
			}
			if c.status != http.StatusOK {
				return
			}
			var dados models.DadosCard
			if err := json.Unmarshal(resp.Dados, &dados); err != nil {
				t.Fatalf("dados inválidos: %v", err)
			}
			codigo := strings.Split(c.caminho, "/")[2]
			if dados.Card.Codigo != codigo || len(calc.cards) != 1 {
				t.Errorf("deveria calcular só %s: %+v (calculados %v)", codigo, dados.Card, calc.cards)
			}
			if calc.filtros.Dashboard != cardsAcessoFalsos[codigo].DashboardChave {
				t.Errorf("filtros normalizados para o dashboard errado: %q", calc.filtros.Dashboard)
			}
		})
	}
}

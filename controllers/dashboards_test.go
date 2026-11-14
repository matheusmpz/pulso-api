package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pulso-api/middlewares"
	"pulso-api/models"
	"pulso-api/utils"
)

// Cenário falso, espelhando o seed: diretoria (1) vê tudo; líder (2) só projetos.
// O dashboard "inativo" existe, mas está desativado.

type repoDashboardFalso struct{ err error }

var dashboardsFalsos = map[string]*models.Dashboard{
	"projetos": {ID: 1, Chave: "projetos", Titulo: "Projetos", Descricao: "P", Ativo: true},
	"vendas":   {ID: 2, Chave: "vendas", Titulo: "Vendas", Descricao: "V", Ativo: true},
	"inativo":  {ID: 9, Chave: "inativo", Titulo: "Inativo", Descricao: "I", Ativo: false},
}

// cardsFalsos: dashboard_id → perfil_id → cards visíveis
var cardsFalsos = map[int64]map[int64][]models.CardMeta{
	1: {
		1: {{Codigo: "proj-01", Titulo: "Progresso", Descricao: "d1", Tipo: "numero"}, {Codigo: "proj-02", Titulo: "Horas", Descricao: "d2", Tipo: "numero"}},
		2: {{Codigo: "proj-01", Titulo: "Progresso", Descricao: "d1", Tipo: "numero"}, {Codigo: "proj-02", Titulo: "Horas", Descricao: "d2", Tipo: "numero"}},
	},
	2: {1: {{Codigo: "ven-01", Titulo: "Receita fechada", Descricao: "d3", Tipo: "numero"}}},
	9: {1: {{Codigo: "x-01", Titulo: "X", Descricao: "dx", Tipo: "numero"}}},
}

func (r *repoDashboardFalso) ListarVisiveis(_ context.Context, perfilID int64) ([]models.DashboardResumo, error) {
	if r.err != nil {
		return nil, r.err
	}
	lista := []models.DashboardResumo{}
	for _, chave := range []string{"projetos", "vendas", "inativo"} {
		d := dashboardsFalsos[chave]
		if n := len(cardsFalsos[d.ID][perfilID]); d.Ativo && n > 0 {
			lista = append(lista, models.DashboardResumo{ID: d.ID, Chave: d.Chave, Titulo: d.Titulo, Descricao: d.Descricao, TotalCards: n})
		}
	}
	return lista, nil
}

func (r *repoDashboardFalso) BuscarPorChave(_ context.Context, chave string) (*models.Dashboard, error) {
	if r.err != nil {
		return nil, r.err
	}
	if d, ok := dashboardsFalsos[chave]; ok {
		return d, nil
	}
	return nil, sql.ErrNoRows
}

type repoCardFalso struct{ err error }

func (r *repoCardFalso) ListarVisiveis(_ context.Context, dashboardID, perfilID int64) ([]models.CardMeta, error) {
	if r.err != nil {
		return nil, r.err
	}
	cards := cardsFalsos[dashboardID][perfilID]
	if cards == nil {
		return []models.CardMeta{}, nil
	}
	return cards, nil
}

func novoRouterDashboards(repo *repoDashboardFalso, cardRepo *repoCardFalso) *gin.Engine {
	gin.SetMode(gin.TestMode)
	ctrl := &DashboardController{Repo: repo, CardRepo: cardRepo}
	r := gin.New()
	g := r.Group("/dashboards", middlewares.AuthMiddleware(cfgJWTTeste))
	g.GET("", ctrl.Listar)
	g.GET("/:chave", ctrl.Buscar)
	return r
}

func tokenDoPerfil(t *testing.T, perfilID int64, perfil string) string {
	t.Helper()
	token, _, err := utils.GerarToken(cfgJWTTeste, "uuid-teste", perfilID, perfil, "t@pulso.local", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestListarDashboards(t *testing.T) {
	casos := []struct {
		nome     string
		perfilID int64
		repoErr  error
		status   int
		chaves   []string
		totais   []int
	}{
		{"diretoria vê projetos e vendas (inativo fora)", 1, nil, http.StatusOK, []string{"projetos", "vendas"}, []int{2, 1}},
		{"líder vê só projetos", 2, nil, http.StatusOK, []string{"projetos"}, []int{2}},
		{"perfil sem cards recebe lista vazia", 3, nil, http.StatusOK, []string{}, []int{}},
		{"erro no banco", 1, errors.New("caiu"), http.StatusInternalServerError, nil, nil},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			r := novoRouterDashboards(&repoDashboardFalso{err: c.repoErr}, &repoCardFalso{})
			status, resp := executar(r, http.MethodGet, "/dashboards", "", tokenDoPerfil(t, c.perfilID, "p"))
			if status != c.status {
				t.Fatalf("status esperado %d, obtido %d (%s)", c.status, status, resp.Erro)
			}
			if c.status != http.StatusOK {
				return
			}

			var dados models.ListaDashboards
			if err := json.Unmarshal(resp.Dados, &dados); err != nil {
				t.Fatalf("dados inválidos: %v", err)
			}
			if dados.Dashboards == nil {
				t.Fatal("a lista deve ser [] e nunca null")
			}
			if len(dados.Dashboards) != len(c.chaves) {
				t.Fatalf("esperados %d dashboards, obtidos %d", len(c.chaves), len(dados.Dashboards))
			}
			for i, d := range dados.Dashboards {
				if d.Chave != c.chaves[i] || d.TotalCards != c.totais[i] {
					t.Errorf("item %d: esperado %s/%d, obtido %s/%d", i, c.chaves[i], c.totais[i], d.Chave, d.TotalCards)
				}
			}
		})
	}
}

func TestBuscarDashboard(t *testing.T) {
	casos := []struct {
		nome     string
		perfilID int64
		chave    string
		repoErr  error
		cardErr  error
		status   int
		mensagem string
		codigos  []string
	}{
		{"diretoria em projetos", 1, "projetos", nil, nil, http.StatusOK, "", []string{"proj-01", "proj-02"}},
		{"líder em projetos", 2, "projetos", nil, nil, http.StatusOK, "", []string{"proj-01", "proj-02"}},
		{"líder em vendas é negado", 2, "vendas", nil, nil, http.StatusForbidden, utils.MsgSemAcesso, nil},
		{"dashboard inexistente", 1, "nao-existe", nil, nil, http.StatusNotFound, msgDashboardNaoEncontrado, nil},
		{"dashboard inativo, mesmo com cards permitidos", 1, "inativo", nil, nil, http.StatusNotFound, msgDashboardNaoEncontrado, nil},
		{"dashboard inativo para quem não teria acesso", 2, "inativo", nil, nil, http.StatusNotFound, msgDashboardNaoEncontrado, nil},
		{"erro ao buscar dashboard", 1, "projetos", errors.New("caiu"), nil, http.StatusInternalServerError, msgErroInterno, nil},
		{"erro ao listar cards", 1, "projetos", nil, errors.New("caiu"), http.StatusInternalServerError, msgErroInterno, nil},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			r := novoRouterDashboards(&repoDashboardFalso{err: c.repoErr}, &repoCardFalso{err: c.cardErr})
			status, resp := executar(r, http.MethodGet, "/dashboards/"+c.chave, "", tokenDoPerfil(t, c.perfilID, "p"))
			if status != c.status {
				t.Fatalf("status esperado %d, obtido %d (%s)", c.status, status, resp.Erro)
			}
			if c.status != http.StatusOK {
				if resp.Erro != c.mensagem {
					t.Errorf("mensagem esperada %q, obtida %q", c.mensagem, resp.Erro)
				}
				return
			}

			var dados models.DashboardDetalhe
			if err := json.Unmarshal(resp.Dados, &dados); err != nil {
				t.Fatalf("dados inválidos: %v", err)
			}
			if dados.Chave != c.chave || len(dados.Cards) != len(c.codigos) {
				t.Fatalf("resposta inesperada: %+v", dados)
			}
			for i, card := range dados.Cards {
				if card.Codigo != c.codigos[i] || card.Descricao == "" || card.Tipo == "" {
					t.Errorf("card %d inesperado: %+v", i, card)
				}
			}
		})
	}
}

func TestDashboardsSemToken(t *testing.T) {
	r := novoRouterDashboards(&repoDashboardFalso{}, &repoCardFalso{})
	for _, caminho := range []string{"/dashboards", "/dashboards/projetos"} {
		if status, _ := executar(r, http.MethodGet, caminho, "", ""); status != http.StatusUnauthorized {
			t.Errorf("%s sem token: esperado 401, obtido %d", caminho, status)
		}
	}
}

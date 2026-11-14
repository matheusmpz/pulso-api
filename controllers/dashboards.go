package controllers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"pulso-api/middlewares"
	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

const msgDashboardNaoEncontrado = "Dashboard não encontrado."

// DashboardController expõe os metadados e os dados dos dashboards,
// sempre filtrados pelo perfil do usuário logado.
type DashboardController struct {
	Repo       repositories.DashboardRepository
	CardRepo   repositories.CardRepository
	FiltroRepo repositories.FiltroRepository
	Calculos   CalculadoraCards
}

// ListaDashboardsDoc é o envelope de sucesso do GET /dashboards (só para o Swagger).
type ListaDashboardsDoc struct {
	Sucesso bool                   `json:"sucesso" example:"true"`
	Dados   models.ListaDashboards `json:"dados"`
}

// DashboardDetalheDoc é o envelope de sucesso do GET /dashboards/:chave (só para o Swagger).
type DashboardDetalheDoc struct {
	Sucesso bool                    `json:"sucesso" example:"true"`
	Dados   models.DashboardDetalhe `json:"dados"`
}

// DadosDashboardDoc é o envelope de sucesso do GET /dashboards/:chave/dados (só para o Swagger).
type DadosDashboardDoc struct {
	Sucesso bool                  `json:"sucesso" example:"true"`
	Dados   models.DadosDashboard `json:"dados"`
}

// Listar godoc
//
// GET /dashboards — dashboards que o perfil pode ver
//
// @Summary      Lista os dashboards visíveis
// @Description  Devolve os dashboards ativos que têm ao menos um card ativo permitido ao perfil do usuário, com a quantidade desses cards (total_cards). Lista vazia se o perfil não vê nenhum.
// @Tags         dashboards
// @Accept       json
// @Produce      json
// @Success      200  {object}  ListaDashboardsDoc
// @Failure      401  {object}  ErroDoc
// @Failure      500  {object}  ErroDoc
// @Security     BearerAuth
// @Router       /dashboards [get]
func (h *DashboardController) Listar(c *gin.Context) {
	perfilID, ok := perfilDoContexto(c)
	if !ok {
		return
	}

	dashboards, err := h.Repo.ListarVisiveis(c.Request.Context(), perfilID)
	if err != nil {
		log.Println("Dashboards: erro ao listar: ", err)
		utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
		return
	}

	utils.RespostaSucesso(c, http.StatusOK, models.ListaDashboards{Dashboards: dashboards})
}

// Buscar godoc
//
// GET /dashboards/:chave — dashboard e a lista de cards visíveis (sem dados)
//
// @Summary      Detalhe de um dashboard
// @Description  Devolve o dashboard e os cards ativos que o perfil pode ver (codigo, titulo, descricao e tipo), ordenados por código e sem os dados calculados. A descricao é o texto do modal de informação.
// @Tags         dashboards
// @Accept       json
// @Produce      json
// @Param        chave  path      string  true  "Chave do dashboard"  Enums(projetos, vendas, ia)
// @Success      200    {object}  DashboardDetalheDoc
// @Failure      401    {object}  ErroDoc
// @Failure      403    {object}  ErroDoc  "Você não tem acesso a este recurso"
// @Failure      404    {object}  ErroDoc  "Dashboard inexistente ou inativo"
// @Failure      500    {object}  ErroDoc
// @Security     BearerAuth
// @Router       /dashboards/{chave} [get]
func (h *DashboardController) Buscar(c *gin.Context) {
	dashboard, cards, ok := h.dashboardVisivel(c)
	if !ok {
		return
	}

	utils.RespostaSucesso(c, http.StatusOK, models.DashboardDetalhe{
		ID:        dashboard.ID,
		Chave:     dashboard.Chave,
		Titulo:    dashboard.Titulo,
		Descricao: dashboard.Descricao,
		Cards:     cards,
	})
}

// Dados godoc
//
// GET /dashboards/:chave/dados — calcula todos os cards visíveis do dashboard
//
// @Summary      Dados de todos os cards de um dashboard
// @Description  Calcula todos os cards que o perfil pode ver no dashboard, para os filtros informados, e devolve a lista de resultados (CardResultado) na ordem dos códigos. Os cards são calculados em paralelo (até 4 ao mesmo tempo, 10 s por card). Se um card falhar, o endpoint não falha: aquele card volta com "erro" no lugar de "dados". Parâmetros que não valem para o dashboard são ignorados (ex.: escopo em vendas). O formato de "dados" depende do tipo do card; veja a seção "Formato dos dados dos cards" do README.
// @Tags         dashboards
// @Accept       json
// @Produce      json
// @Param        chave           path      string  true   "Chave do dashboard"  Enums(projetos, vendas, ia)
// @Param        escopo          query     string  false  "Escopo (projetos e ia)"  Enums(projeto, time, sprint)  default(projeto)
// @Param        id              query     int     false  "Id do projeto, time ou sprint, conforme o escopo (obrigatório em escopo=sprint)"
// @Param        de              query     string  false  "Início do período (AAAA-MM-DD). Padrão: 29 dias antes de ate"
// @Param        ate             query     string  false  "Fim do período (AAAA-MM-DD). Padrão: hoje"
// @Param        tipo_projeto    query     string  false  "Tipos de projeto, separados por vírgula (web, mobile, integracao, api, outro)"
// @Param        status_projeto  query     string  false  "Status de projeto, separados por vírgula (projetos e ia): em_andamento, concluido, pausado, cancelado"
// @Param        origem_lead     query     string  false  "Origens de lead, separadas por vírgula (vendas)"
// @Param        modelo          query     string  false  "Modelos de IA, separados por vírgula (ia)"
// @Success      200             {object}  DadosDashboardDoc
// @Failure      400             {object}  ErroDoc  "Filtro inválido"
// @Failure      401             {object}  ErroDoc
// @Failure      403             {object}  ErroDoc  "Você não tem acesso a este recurso"
// @Failure      404             {object}  ErroDoc  "Dashboard inexistente ou inativo"
// @Failure      500             {object}  ErroDoc
// @Security     BearerAuth
// @Router       /dashboards/{chave}/dados [get]
func (h *DashboardController) Dados(c *gin.Context) {
	dashboard, cards, ok := h.dashboardVisivel(c)
	if !ok {
		return
	}

	filtros, ok := filtrosDaRequisicao(c, dashboard.Chave, h.FiltroRepo)
	if !ok {
		return
	}

	utils.RespostaSucesso(c, http.StatusOK, models.DadosDashboard{
		Dashboard:        models.DashboardRef{ID: dashboard.ID, Chave: dashboard.Chave, Titulo: dashboard.Titulo},
		FiltrosAplicados: filtros,
		Cards:            h.Calculos.CalcularCards(c.Request.Context(), cards, filtros),
	})
}

// dashboardVisivel busca o dashboard da rota e os cards que o perfil vê
// nele, aplicando a regra de acesso. Em caso de 401/403/404/500 já
// responde e devolve ok = false.
func (h *DashboardController) dashboardVisivel(c *gin.Context) (*models.Dashboard, []models.CardMeta, bool) {
	perfilID, ok := perfilDoContexto(c)
	if !ok {
		return nil, nil, false
	}
	ctx := c.Request.Context()

	dashboard, err := h.Repo.BuscarPorChave(ctx, c.Param("chave"))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Println("Dashboards: erro ao buscar dashboard: ", err)
		utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
		return nil, nil, false
	}
	existe := err == nil

	var cards []models.CardMeta
	if existe && dashboard.Ativo {
		cards, err = h.CardRepo.ListarVisiveis(ctx, dashboard.ID, perfilID)
		if err != nil {
			log.Println("Dashboards: erro ao listar cards: ", err)
			utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
			return nil, nil, false
		}
	}

	// O dashboard é permitido quando o perfil vê ao menos um card ativo dele
	acesso := utils.AvaliarAcesso(existe, existe && dashboard.Ativo, len(cards) > 0)
	switch acesso {
	case utils.AcessoNaoEncontrado:
		utils.RespostaErro(c, acesso.StatusHTTP(), msgDashboardNaoEncontrado)
		return nil, nil, false
	case utils.AcessoNegado:
		utils.RespostaErro(c, acesso.StatusHTTP(), utils.MsgSemAcesso)
		return nil, nil, false
	}
	return dashboard, cards, true
}

// perfilDoContexto lê o perfil_id gravado pelo AuthMiddleware. Se não
// houver, já responde 401 e devolve ok = false.
func perfilDoContexto(c *gin.Context) (int64, bool) {
	perfilID := c.GetInt64(middlewares.ChavePerfilID)
	if perfilID <= 0 {
		utils.RespostaErro(c, http.StatusUnauthorized, msgSessaoInvalida)
		return 0, false
	}
	return perfilID, true
}

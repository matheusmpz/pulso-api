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

// DashboardController expõe os metadados dos dashboards e dos cards,
// sempre filtrados pelo perfil do usuário logado.
type DashboardController struct {
	Repo     repositories.DashboardRepository
	CardRepo repositories.CardRepository
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
	perfilID, ok := perfilDoContexto(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()

	dashboard, err := h.Repo.BuscarPorChave(ctx, c.Param("chave"))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Println("Dashboards: erro ao buscar dashboard: ", err)
		utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
		return
	}
	existe := err == nil

	var cards []models.CardMeta
	if existe && dashboard.Ativo {
		cards, err = h.CardRepo.ListarVisiveis(ctx, dashboard.ID, perfilID)
		if err != nil {
			log.Println("Dashboards: erro ao listar cards: ", err)
			utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
			return
		}
	}

	// O dashboard é permitido quando o perfil vê ao menos um card ativo dele
	acesso := utils.AvaliarAcesso(existe, existe && dashboard.Ativo, len(cards) > 0)
	switch acesso {
	case utils.AcessoNaoEncontrado:
		utils.RespostaErro(c, acesso.StatusHTTP(), msgDashboardNaoEncontrado)
		return
	case utils.AcessoNegado:
		utils.RespostaErro(c, acesso.StatusHTTP(), utils.MsgSemAcesso)
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

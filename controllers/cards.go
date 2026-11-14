package controllers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

const msgCardNaoEncontrado = "Card não encontrado."

// CardController calcula um card isolado.
type CardController struct {
	Repo       repositories.CardRepository
	FiltroRepo repositories.FiltroRepository
	Calculos   CalculadoraCards
}

// DadosCardDoc é o envelope de sucesso do GET /cards/:codigo/dados (só para o Swagger).
type DadosCardDoc struct {
	Sucesso bool             `json:"sucesso" example:"true"`
	Dados   models.DadosCard `json:"dados"`
}

// Dados godoc
//
// GET /cards/:codigo/dados — calcula um card
//
// @Summary      Dados de um card
// @Description  Calcula um card para os filtros informados. Os filtros valem conforme o dashboard do card (ex.: escopo é ignorado nos cards de vendas). Se o cálculo falhar, a resposta continua 200 e o card volta com "erro" no lugar de "dados", igual ao GET /dashboards/{chave}/dados. O formato de "dados" depende do tipo do card; veja a seção "Formato dos dados dos cards" do README.
// @Tags         cards
// @Accept       json
// @Produce      json
// @Param        codigo          path      string  true   "Código do card (ex.: proj-01, ven-03, ia-12)"
// @Param        escopo          query     string  false  "Escopo (projetos e ia)"  Enums(projeto, time, sprint)  default(projeto)
// @Param        id              query     int     false  "Id do projeto, time ou sprint, conforme o escopo (obrigatório em escopo=sprint)"
// @Param        de              query     string  false  "Início do período (AAAA-MM-DD). Padrão: 29 dias antes de ate"
// @Param        ate             query     string  false  "Fim do período (AAAA-MM-DD). Padrão: hoje"
// @Param        tipo_projeto    query     string  false  "Tipos de projeto, separados por vírgula (web, mobile, integracao, api, outro)"
// @Param        status_projeto  query     string  false  "Status de projeto, separados por vírgula (projetos e ia)"
// @Param        origem_lead     query     string  false  "Origens de lead, separadas por vírgula (vendas)"
// @Param        modelo          query     string  false  "Modelos de IA, separados por vírgula (ia)"
// @Success      200             {object}  DadosCardDoc
// @Failure      400             {object}  ErroDoc  "Filtro inválido"
// @Failure      401             {object}  ErroDoc
// @Failure      403             {object}  ErroDoc  "Você não tem acesso a este recurso"
// @Failure      404             {object}  ErroDoc  "Card inexistente ou inativo"
// @Failure      500             {object}  ErroDoc
// @Security     BearerAuth
// @Router       /cards/{codigo}/dados [get]
func (h *CardController) Dados(c *gin.Context) {
	perfilID, ok := perfilDoContexto(c)
	if !ok {
		return
	}

	card, err := h.Repo.BuscarAcesso(c.Request.Context(), c.Param("codigo"), perfilID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Println("Cards: erro ao buscar card: ", err)
		utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
		return
	}
	existe := err == nil

	acesso := utils.AvaliarAcesso(existe, existe && card.Ativo, existe && card.Permitido)
	switch acesso {
	case utils.AcessoNaoEncontrado:
		utils.RespostaErro(c, acesso.StatusHTTP(), msgCardNaoEncontrado)
		return
	case utils.AcessoNegado:
		utils.RespostaErro(c, acesso.StatusHTTP(), utils.MsgSemAcesso)
		return
	}

	filtros, ok := filtrosDaRequisicao(c, card.DashboardChave, h.FiltroRepo)
	if !ok {
		return
	}

	utils.RespostaSucesso(c, http.StatusOK, models.DadosCard{
		FiltrosAplicados: filtros,
		Card:             h.Calculos.CalcularCard(c.Request.Context(), card.CardMeta, filtros),
	})
}

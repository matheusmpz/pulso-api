package controllers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// FiltroController expõe as opções para preencher os filtros do front.
type FiltroController struct {
	Repo repositories.FiltroRepository
	// Hoje devolve o dia atual no fuso do app (substituível nos testes).
	Hoje func() models.Data
}

// OpcoesFiltrosDoc é o envelope de sucesso do GET /filtros/opcoes (só para o Swagger).
type OpcoesFiltrosDoc struct {
	Sucesso bool                 `json:"sucesso" example:"true"`
	Dados   models.OpcoesFiltros `json:"dados"`
}

// Opcoes godoc
//
// GET /filtros/opcoes — opções para preencher os filtros
//
// @Summary      Opções dos filtros
// @Description  Devolve as opções para preencher os filtros: tipos e status de projeto (valor e rótulo), origens de lead e modelos de IA existentes, projetos, times e sprints. Em cada sprint, "atual" indica se hoje (no fuso do app) está entre o início e o fim; o front usa isso para pré-selecionar a sprint atual.
// @Tags         filtros
// @Accept       json
// @Produce      json
// @Success      200  {object}  OpcoesFiltrosDoc
// @Failure      401  {object}  ErroDoc
// @Failure      500  {object}  ErroDoc
// @Security     BearerAuth
// @Router       /filtros/opcoes [get]
func (h *FiltroController) Opcoes(c *gin.Context) {
	ctx := c.Request.Context()
	falhou := func(oque string, err error) {
		log.Printf("Filtros: erro ao listar %s: %v", oque, err)
		utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
	}

	origens, err := h.Repo.Origens(ctx)
	if err != nil {
		falhou("origens", err)
		return
	}
	modelos, err := h.Repo.Modelos(ctx)
	if err != nil {
		falhou("modelos", err)
		return
	}
	projetos, err := h.Repo.Projetos(ctx)
	if err != nil {
		falhou("projetos", err)
		return
	}
	times, err := h.Repo.Times(ctx)
	if err != nil {
		falhou("times", err)
		return
	}
	sprints, err := h.Repo.Sprints(ctx)
	if err != nil {
		falhou("sprints", err)
		return
	}

	utils.RespostaSucesso(c, http.StatusOK, models.OpcoesFiltros{
		TiposProjeto:  models.TiposProjeto,
		StatusProjeto: models.StatusProjeto,
		Origens:       origens,
		Modelos:       modelos,
		Projetos:      projetos,
		Times:         times,
		Sprints:       marcarSprintAtual(sprints, h.hoje()),
	})
}

func (h *FiltroController) hoje() models.Data {
	if h.Hoje != nil {
		return h.Hoje()
	}
	return utils.Hoje()
}

// marcarSprintAtual marca as sprints em que hoje está entre o início e o
// fim (inclusive).
func marcarSprintAtual(sprints []models.Sprint, hoje models.Data) []models.SprintOpcao {
	opcoes := make([]models.SprintOpcao, len(sprints))
	for i, s := range sprints {
		atual := !hoje.Before(s.Inicio.Time) && !hoje.After(s.Fim.Time)
		opcoes[i] = models.SprintOpcao{Sprint: s, Atual: atual}
	}
	return opcoes
}

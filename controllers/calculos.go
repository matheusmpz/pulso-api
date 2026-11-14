package controllers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// CalculadoraCards é o que os controllers precisam do serviço de cálculo
// (o *calculos.Servico satisfaz).
type CalculadoraCards interface {
	CalcularCards(ctx context.Context, cards []models.CardMeta, f models.Filtros) []models.CardResultado
	CalcularCard(ctx context.Context, card models.CardMeta, f models.Filtros) models.CardResultado
}

// filtrosDaRequisicao lê, valida e normaliza os filtros da query string
// para o dashboard. Em erro de validação responde 400; em erro de banco
// (busca da sprint), 500. Devolve ok = false se já respondeu.
func filtrosDaRequisicao(c *gin.Context, dashboard string, filtroRepo repositories.FiltroRepository) (models.Filtros, bool) {
	entrada, err := utils.EntradaDeQuery(c.Request.URL.Query())
	if err == nil {
		var f models.Filtros
		f, err = utils.NormalizarFiltros(c.Request.Context(), entrada, dashboard, utils.Hoje(), filtroRepo.BuscarSprint)
		if err == nil {
			return f, true
		}
	}

	var ef *utils.ErroFiltro
	if errors.As(err, &ef) {
		utils.RespostaErro(c, http.StatusBadRequest, ef.Mensagem)
		return models.Filtros{}, false
	}
	log.Println("Filtros: erro ao normalizar: ", err)
	utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
	return models.Filtros{}, false
}

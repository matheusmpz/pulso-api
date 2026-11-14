package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// tempoPingHealth é o prazo para o banco responder no /health.
const tempoPingHealth = 2 * time.Second

// Pinger é o que o health precisa do banco (o *sql.DB satisfaz).
type Pinger interface {
	PingContext(ctx context.Context) error
}

// HealthController verifica se a API e o banco estão no ar.
type HealthController struct {
	DB Pinger
}

// HealthResposta é o corpo do GET /health.
type HealthResposta struct {
	Status string `json:"status" example:"ok"`
	Banco  string `json:"banco" example:"ok"`
}

// Verificar godoc
//
// GET /health — informa se a API está no ar e se o banco responde
//
// @Summary      Verifica a saúde da API
// @Description  Devolve o status da API e do banco. Responde 503 se o banco não responder ao ping.
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  HealthResposta
// @Failure      503  {object}  HealthResposta
// @Router       /health [get]
func (h *HealthController) Verificar(c *gin.Context) {
	ctx, cancelar := context.WithTimeout(c.Request.Context(), tempoPingHealth)
	defer cancelar()

	if err := h.DB.PingContext(ctx); err != nil {
		log.Println("Health: banco indisponível: ", err)
		c.JSON(http.StatusServiceUnavailable, HealthResposta{Status: "erro", Banco: "indisponivel"})
		return
	}

	c.JSON(http.StatusOK, HealthResposta{Status: "ok", Banco: "ok"})
}

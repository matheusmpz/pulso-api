package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// pingerFalso simula o banco no teste.
type pingerFalso struct{ err error }

func (p pingerFalso) PingContext(context.Context) error { return p.err }

func TestHealthVerificar(t *testing.T) {
	gin.SetMode(gin.TestMode)

	casos := []struct {
		nome     string
		errPing  error
		status   int
		esperado HealthResposta
	}{
		{"banco ok", nil, http.StatusOK, HealthResposta{Status: "ok", Banco: "ok"}},
		{"banco fora", errors.New("conexão recusada"), http.StatusServiceUnavailable, HealthResposta{Status: "erro", Banco: "indisponivel"}},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			ctrl := &HealthController{DB: pingerFalso{err: c.errPing}}
			r := gin.New()
			r.GET("/health", ctrl.Verificar)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

			if w.Code != c.status {
				t.Fatalf("status esperado %d, obtido %d", c.status, w.Code)
			}
			var obtido HealthResposta
			if err := json.Unmarshal(w.Body.Bytes(), &obtido); err != nil {
				t.Fatalf("corpo inválido: %v", err)
			}
			if obtido != c.esperado {
				t.Errorf("esperado %+v, obtido %+v", c.esperado, obtido)
			}
		})
	}
}

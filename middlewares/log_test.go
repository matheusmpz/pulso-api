package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOcultarToken(t *testing.T) {
	casos := map[string]string{
		"/ws?token=eyJhbGciOi.abc.def":           "/ws?token=***",
		"/ws?x=1&token=segredo&y=2":              "/ws?x=1&token=***&y=2",
		"/dashboards/projetos/dados?escopo=time": "/dashboards/projetos/dados?escopo=time",
		"/ws?tokenizar=nao":                      "/ws?tokenizar=nao",
		"/health":                                "/health",
	}
	for entrada, esperado := range casos {
		if got := OcultarToken(entrada); got != esperado {
			t.Errorf("OcultarToken(%q): esperado %q, obtido %q", entrada, esperado, got)
		}
	}
}

func TestLoggerNaoGravaToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var saida bytes.Buffer
	antes := gin.DefaultWriter
	gin.DefaultWriter = &saida
	defer func() { gin.DefaultWriter = antes }()

	r := gin.New()
	r.Use(Logger())
	r.GET("/ws", func(c *gin.Context) { c.Status(http.StatusUnauthorized) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ws?token=eyJsegredo.muito.secreto", nil))

	if strings.Contains(saida.String(), "segredo") {
		t.Fatalf("o token apareceu no log: %s", saida.String())
	}
	if !strings.Contains(saida.String(), "/ws?token=***") || !strings.Contains(saida.String(), "401") {
		t.Errorf("linha de log inesperada: %s", saida.String())
	}
}

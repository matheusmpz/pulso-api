package middlewares

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

// tokenNaQuery encontra o valor do parâmetro token (ex.: /ws?token=eyJ...).
var tokenNaQuery = regexp.MustCompile(`([?&]token=)[^&\s]*`)

// OcultarToken troca o valor do parâmetro token da URL por "***".
func OcultarToken(caminho string) string {
	return tokenNaQuery.ReplaceAllString(caminho, "${1}***")
}

// Logger é o log de requisições do Gin, no mesmo formato do padrão, mas
// sem o token: o WebSocket recebe o JWT na query (?token=) e ele nunca
// pode ir para o log.
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		return fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %q\n%s",
			p.TimeStamp.Format("2006/01/02 - 15:04:05"),
			p.StatusCode,
			p.Latency.Truncate(time.Microsecond),
			p.ClientIP,
			p.Method,
			OcultarToken(p.Path),
			p.ErrorMessage,
		)
	})
}

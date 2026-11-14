package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"pulso-api/utils"
)

var (
	db   *sql.DB   // pool de conexões, preenchido uma única vez dentro do once.Do
	once sync.Once // garante que o bloco de inicialização rode só uma vez
)

// tempoPingInicial é o prazo para o banco responder na inicialização.
const tempoPingInicial = 5 * time.Second

// GetDB devolve o pool de conexões, criando-o na primeira chamada.
// As variáveis de ambiente já devem estar carregadas (main faz o
// godotenv.Load). Qualquer falha encerra a aplicação com mensagem clara.
func GetDB() *sql.DB {
	once.Do(func() {
		connStr := montarConnString(map[string]string{
			"host":     utils.EnvOuPadrao("DB_HOST", ""),
			"port":     utils.EnvOuPadrao("DB_PORT", "5432"),
			"user":     utils.EnvOuPadrao("DB_USER", ""),
			"password": utils.EnvOuPadrao("DB_PASSWORD", ""),
			"dbname":   utils.EnvOuPadrao("DB_NAME", "pulso_db"),
			"sslmode":  utils.EnvOuPadrao("DB_SSLMODE", "disable"),
		})

		// sql.Open não conecta: só valida a string e prepara o driver
		conn, err := sql.Open("postgres", connStr)
		if err != nil {
			log.Fatal("Erro ao abrir conexão com o banco: ", err)
		}

		// O *sql.DB é um pool. Sem limites ele abre conexões ilimitadas
		// e pode derrubar o Postgres.
		conn.SetMaxOpenConns(25)                 // máximo de conexões simultâneas
		conn.SetMaxIdleConns(10)                 // conexões ociosas reaproveitáveis
		conn.SetConnMaxLifetime(5 * time.Minute) // vida máxima de uma conexão
		conn.SetConnMaxIdleTime(2 * time.Minute) // tempo máximo ociosa no pool

		// Faz a conexão de fato
		ctx, cancelar := context.WithTimeout(context.Background(), tempoPingInicial)
		defer cancelar()
		if err := conn.PingContext(ctx); err != nil {
			log.Fatal("Erro ao conectar no banco (confira as variáveis DB_*): ", err)
		}

		db = conn
	})

	return db
}

// ordemChaves fixa a ordem dos parâmetros na string de conexão.
var ordemChaves = []string{"host", "port", "user", "password", "dbname", "sslmode"}

// montarConnString monta a string no formato chave='valor' do lib/pq,
// escapando aspas simples e barras invertidas. Assim uma senha com espaço
// ou aspas não quebra a conexão. Chaves com valor vazio são omitidas.
func montarConnString(params map[string]string) string {
	partes := make([]string, 0, len(ordemChaves))
	for _, chave := range ordemChaves {
		valor, ok := params[chave]
		if !ok || valor == "" {
			continue
		}
		escapado := strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(valor)
		partes = append(partes, fmt.Sprintf("%s='%s'", chave, escapado))
	}
	return strings.Join(partes, " ")
}

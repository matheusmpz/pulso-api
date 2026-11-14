package db

import "testing"

func TestMontarConnString(t *testing.T) {
	casos := []struct {
		nome     string
		params   map[string]string
		esperado string
	}{
		{
			nome: "valores simples",
			params: map[string]string{
				"host": "localhost", "port": "5432", "user": "pulso",
				"password": "segredo", "dbname": "pulso_db", "sslmode": "disable",
			},
			esperado: "host='localhost' port='5432' user='pulso' password='segredo' dbname='pulso_db' sslmode='disable'",
		},
		{
			nome:     "senha com espaço, aspas e barra",
			params:   map[string]string{"host": "db", "password": `a b'c\d`},
			esperado: `host='db' password='a b\'c\\d'`,
		},
		{
			nome:     "valores vazios são omitidos",
			params:   map[string]string{"host": "db", "user": "", "dbname": "pulso_db"},
			esperado: "host='db' dbname='pulso_db'",
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if obtido := montarConnString(c.params); obtido != c.esperado {
				t.Errorf("esperado %q, obtido %q", c.esperado, obtido)
			}
		})
	}
}

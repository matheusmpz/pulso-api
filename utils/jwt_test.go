package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var cfgTeste = ConfigJWT{Segredo: []byte("segredo-de-teste"), Duracao: time.Hour}

func TestGerarEValidarToken(t *testing.T) {
	agora := time.Now()
	token, expiraEm, err := GerarToken(cfgTeste, "uuid-1", 2, "lider_tecnologia", "lider@pulso.local", agora)
	if err != nil {
		t.Fatalf("erro ao gerar: %v", err)
	}
	esperada := agora.Truncate(time.Second).Add(time.Hour)
	if !expiraEm.Equal(esperada) {
		t.Errorf("expiração esperada %v, obtida %v", esperada, expiraEm)
	}

	claims, err := ValidarToken(cfgTeste, token)
	if err != nil {
		t.Fatalf("token válido rejeitado: %v", err)
	}
	if claims.UsuarioID != "uuid-1" || claims.PerfilID != 2 || claims.Perfil != "lider_tecnologia" || claims.Email != "lider@pulso.local" {
		t.Errorf("claims inesperados: %+v", claims)
	}
}

func TestValidarTokenRejeita(t *testing.T) {
	valido, _, _ := GerarToken(cfgTeste, "uuid-1", 1, "diretoria", "d@pulso.local", time.Now())
	expirado, _, _ := GerarToken(cfgTeste, "uuid-1", 1, "diretoria", "d@pulso.local", time.Now().Add(-2*time.Hour))
	outroSegredo, _, _ := GerarToken(ConfigJWT{Segredo: []byte("outro"), Duracao: time.Hour}, "uuid-1", 1, "diretoria", "d@pulso.local", time.Now())

	// Token assinado com outro algoritmo (HS512) e mesmo segredo
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{
		UsuarioID: "uuid-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    emissorToken,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString(cfgTeste.Segredo)

	// Token sem expiração
	semExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UsuarioID:        "uuid-1",
		RegisteredClaims: jwt.RegisteredClaims{Issuer: emissorToken},
	}).SignedString(cfgTeste.Segredo)

	// Token de outro emissor
	outroEmissor, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UsuarioID: "uuid-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "outra-api",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString(cfgTeste.Segredo)

	casos := []struct {
		nome  string
		token string
	}{
		{"vazio", ""},
		{"malformado", "nao.e.jwt"},
		{"expirado", expirado},
		{"outro segredo", outroSegredo},
		{"algoritmo diferente de HS256", hs512},
		{"sem expiração", semExp},
		{"outro emissor", outroEmissor},
		{"adulterado", valido + "x"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if _, err := ValidarToken(cfgTeste, c.token); err == nil {
				t.Error("token deveria ser rejeitado")
			}
		})
	}
}

func TestParseHorasExpiracao(t *testing.T) {
	casos := []struct {
		valor    string
		esperado time.Duration
		erro     bool
	}{
		{"", 24 * time.Hour, false},
		{"24", 24 * time.Hour, false},
		{" 8 ", 8 * time.Hour, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"1.5", 0, true},
	}

	for _, c := range casos {
		t.Run(c.valor, func(t *testing.T) {
			obtido, err := ParseHorasExpiracao(c.valor)
			if (err != nil) != c.erro {
				t.Fatalf("erro esperado=%v, obtido %v", c.erro, err)
			}
			if obtido != c.esperado {
				t.Errorf("esperado %v, obtido %v", c.esperado, obtido)
			}
		})
	}
}

func TestCarregarConfigJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_EXPIRA_HORAS", "")
	if _, err := CarregarConfigJWT(); err == nil {
		t.Error("sem JWT_SECRET deveria dar erro")
	}

	t.Setenv("JWT_SECRET", "abc")
	t.Setenv("JWT_EXPIRA_HORAS", "12")
	cfg, err := CarregarConfigJWT()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if string(cfg.Segredo) != "abc" || cfg.Duracao != 12*time.Hour {
		t.Errorf("config inesperada: %+v", cfg)
	}
}

// Package authn emite e valida o JWT próprio do Meirinho.
//
// Optamos por JWT HS256 assinado por nós em vez de provedor externo: no MVP
// não vale acoplar a infra a um terceiro. O claim que importa é o tenant_id —
// ele é a chave de todo o isolamento de dados.
package authn

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "meirinho"

var (
	ErrTokenInvalido = errors.New("token inválido")
	ErrSemToken      = errors.New("token ausente")
)

// Claims carrega o tenant. Subject == tenant_id: no MVP um advogado é um
// tenant, então não há usuário separado do tenant ainda.
type Claims struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Nome     string `json:"nome"`
	jwt.RegisteredClaims
}

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

func (i *Issuer) Emitir(tenantID, email, nome string) (string, time.Time, error) {
	exp := time.Now().Add(i.ttl)
	c := Claims{
		TenantID: tenantID,
		Email:    email,
		Nome:     nome,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   tenantID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return tok, exp, nil
}

type Verifier struct{ secret []byte }

func NewVerifier(secret string) *Verifier { return &Verifier{secret: []byte(secret)} }

func (v *Verifier) Verificar(token string) (*Claims, error) {
	var c Claims
	_, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", t.Header["alg"])
		}
		return v.secret, nil
	}, jwt.WithIssuer(issuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, ErrTokenInvalido
	}
	if c.TenantID == "" {
		return nil, ErrTokenInvalido
	}
	return &c, nil
}

// BearerDoRequest extrai o token do header Authorization.
func BearerDoRequest(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", ErrSemToken
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrTokenInvalido
	}
	return strings.TrimSpace(parts[1]), nil
}

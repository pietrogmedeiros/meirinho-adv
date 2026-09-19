package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pietromedeiros/meirinho/internal/platform/db"
)

var (
	errEmailEmUso         = errors.New("email já cadastrado")
	errCredencialInvalida = errors.New("credenciais inválidas")
)

type tenant struct {
	ID           string    `json:"id"`
	Nome         string    `json:"nome"`
	OAB          string    `json:"oab"`
	Email        string    `json:"email"`
	RetencaoDias int       `json:"retencao_dias"`
	CreatedAt    time.Time `json:"created_at"`
}

type store struct{ pool *db.Pool }

// criar insere o tenant. A unicidade do e-mail é do banco (índice único em
// lower(email)) — checar antes seria uma corrida esperando acontecer.
func (s *store) criar(ctx context.Context, nome, oab, email, senhaHash string) (*tenant, error) {
	var t tenant
	err := s.pool.AdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO auth.tenants (nome, oab, email, senha_hash)
			VALUES ($1, $2, $3, $4)
			RETURNING id, nome, oab, email, retencao_dias, created_at`,
			nome, oab, strings.TrimSpace(email), senhaHash,
		).Scan(&t.ID, &t.Nome, &t.OAB, &t.Email, &t.RetencaoDias, &t.CreatedAt)
	})
	if err != nil {
		if strings.Contains(err.Error(), "tenants_email_uniq") {
			return nil, errEmailEmUso
		}
		return nil, fmt.Errorf("criar tenant: %w", err)
	}
	return &t, nil
}

func (s *store) porEmail(ctx context.Context, email string) (*tenant, string, error) {
	var t tenant
	var hash string
	err := s.pool.AdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, nome, oab, email, retencao_dias, created_at, senha_hash
			  FROM auth.tenants
			 WHERE lower(email) = lower($1)`, strings.TrimSpace(email),
		).Scan(&t.ID, &t.Nome, &t.OAB, &t.Email, &t.RetencaoDias, &t.CreatedAt, &hash)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", errCredencialInvalida
	}
	if err != nil {
		return nil, "", fmt.Errorf("buscar tenant: %w", err)
	}
	return &t, hash, nil
}

func (s *store) porID(ctx context.Context, id string) (*tenant, error) {
	var t tenant
	err := s.pool.AdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, nome, oab, email, retencao_dias, created_at
			  FROM auth.tenants WHERE id = $1`, id,
		).Scan(&t.ID, &t.Nome, &t.OAB, &t.Email, &t.RetencaoDias, &t.CreatedAt)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errCredencialInvalida
	}
	if err != nil {
		return nil, fmt.Errorf("buscar tenant: %w", err)
	}
	return &t, nil
}

// atualizarRetencao é o controle de política de retenção (LGPD) do tenant.
func (s *store) atualizarRetencao(ctx context.Context, id string, dias int) (*tenant, error) {
	var t tenant
	err := s.pool.AdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			UPDATE auth.tenants SET retencao_dias = $2 WHERE id = $1
			RETURNING id, nome, oab, email, retencao_dias, created_at`, id, dias,
		).Scan(&t.ID, &t.Nome, &t.OAB, &t.Email, &t.RetencaoDias, &t.CreatedAt)
	})
	if err != nil {
		return nil, fmt.Errorf("atualizar retenção: %w", err)
	}
	return &t, nil
}

func (s *store) atualizarPerfil(ctx context.Context, id, nome, oab, email string) (*tenant, error) {
	var t tenant
	err := s.pool.AdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			UPDATE auth.tenants SET nome = $2, oab = $3, email = $4 WHERE id = $1
			RETURNING id, nome, oab, email, retencao_dias, created_at`, id, nome, oab, email,
		).Scan(&t.ID, &t.Nome, &t.OAB, &t.Email, &t.RetencaoDias, &t.CreatedAt)
	})
	if err != nil {
		if strings.Contains(err.Error(), "tenants_email_uniq") {
			return nil, errEmailEmUso
		}
		return nil, fmt.Errorf("atualizar perfil: %w", err)
	}
	return &t, nil
}

func (s *store) senhaHash(ctx context.Context, id string) (string, error) {
	var hash string
	err := s.pool.AdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT senha_hash FROM auth.tenants WHERE id = $1`, id).Scan(&hash)
	})
	return hash, err
}

func (s *store) atualizarSenha(ctx context.Context, id, hash string) error {
	return s.pool.AdminTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE auth.tenants SET senha_hash = $2 WHERE id = $1`, id, hash)
		return err
	})
}

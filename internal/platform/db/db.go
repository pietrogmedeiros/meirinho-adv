// Package db entrega o pool Postgres e — mais importante — o único caminho
// pelo qual os serviços leem/escrevem dado de tenant.
//
// O isolamento não é responsabilidade só da aplicação: as tabelas têm
// ROW LEVEL SECURITY ligada e as políticas comparam `tenant_id` com a variável
// de sessão `app.current_tenant`. Quem esquecer de filtrar no SQL simplesmente
// não enxerga a linha. TenantTx é quem seta essa variável.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSemTenant = errors.New("db: tenant_id vazio — recusando abrir transação")

type Pool struct{ *pgxpool.Pool }

func Open(ctx context.Context, dsn string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("dsn inválido: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour

	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Pool{p}, nil
}

// Aguardar dá tempo do Postgres subir antes do serviço desistir. No compose os
// containers sobem juntos e o healthcheck nem sempre basta.
func (p *Pool) Aguardar(ctx context.Context, tentativas int) error {
	var last error
	for i := 0; i < tentativas; i++ {
		if err := p.Ping(ctx); err == nil {
			return nil
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("postgres indisponível: %w", last)
}

// TenantTx abre uma transação com `app.current_tenant` setado, roda fn e
// commita. Toda leitura/escrita de dado de advogado passa por aqui.
func (p *Pool) TenantTx(ctx context.Context, tenantID string, fn func(pgx.Tx) error) error {
	if tenantID == "" {
		return ErrSemTenant
	}
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// set_config com parâmetro: nunca interpolar tenant_id em SQL.
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// AdminTx roda fora do escopo de tenant (cadastro, login — momentos em que
// ainda não se sabe qual é o tenant). As tabelas tocadas aqui são só as de
// `auth`, que não têm RLS por definição.
func (p *Pool) AdminTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

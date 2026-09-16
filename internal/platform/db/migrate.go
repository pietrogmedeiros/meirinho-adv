package db

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed all:migrations
var migrationsFS embed.FS

// Migrar aplica os arquivos .sql de migrations/ em ordem, uma vez cada.
// Deliberadamente simples: sem down-migration, sem ferramenta externa. Cada
// arquivo roda dentro de uma transação — falhou, nada foi aplicado.
func Migrar(ctx context.Context, p *Pool, log *slog.Logger) error {
	_, err := p.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			versao     TEXT PRIMARY KEY,
			aplicada_em TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return fmt.Errorf("criar schema_migrations: %w", err)
	}

	entradas, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("ler migrations embutidas: %w", err)
	}

	var nomes []string
	for _, e := range entradas {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			nomes = append(nomes, e.Name())
		}
	}
	sort.Strings(nomes)

	for _, nome := range nomes {
		var existe bool
		if err := p.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE versao = $1)`, nome,
		).Scan(&existe); err != nil {
			return fmt.Errorf("checar migration %s: %w", nome, err)
		}
		if existe {
			log.Debug("migration já aplicada", "versao", nome)
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + nome)
		if err != nil {
			return fmt.Errorf("ler %s: %w", nome, err)
		}

		if err := p.AdminTx(ctx, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
				return fmt.Errorf("executar %s: %w", nome, err)
			}
			_, err := tx.Exec(ctx, `INSERT INTO schema_migrations (versao) VALUES ($1)`, nome)
			return err
		}); err != nil {
			return err
		}
		log.Info("migration aplicada", "versao", nome)
	}
	return nil
}

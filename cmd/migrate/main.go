// Binário de migration. Roda uma vez no boot do compose, antes dos serviços,
// conectado como dono do banco (os serviços conectam como meirinho_app).
package main

import (
	"context"
	"os"
	"time"

	"github.com/pietromedeiros/meirinho/internal/platform/config"
	"github.com/pietromedeiros/meirinho/internal/platform/db"
	"github.com/pietromedeiros/meirinho/internal/platform/logging"
)

func main() {
	log := logging.New("migrate", config.String("LOG_LEVEL", "info"))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := db.Open(ctx, config.MustString("DATABASE_URL"))
	if err != nil {
		log.Error("abrir banco", "err", err)
		panic(err)
	}
	defer pool.Close()

	if err := pool.Aguardar(ctx, 60); err != nil {
		// A causa mais comum em servidor: o volume do Postgres foi criado com
		// outra POSTGRES_PASSWORD (ela só vale na primeira subida do volume).
		log.Error("conectar ao postgres; se for falha de autenticação, a POSTGRES_PASSWORD "+
			"não é a mesma com que o volume foi criado", "err", err)
		os.Exit(1)
	}
	if err := db.Migrar(ctx, pool, log); err != nil {
		log.Error("migrar", "err", err)
		os.Exit(1)
	}
	if err := sincronizarPapelApp(ctx, pool, config.String("APP_DB_PASSWORD", "")); err != nil {
		log.Error("sincronizar papel meirinho_app", "err", err)
		os.Exit(1)
	}
	log.Info("migrations em dia")
}

// sincronizarPapelApp garante que o papel dos serviços exista com a senha de
// APP_DB_PASSWORD. O script de init do Postgres só roda na criação do volume;
// sem isto, trocar a variável depois (ou definir só depois do primeiro deploy)
// deixaria os serviços sem conseguir conectar, sem nenhum caminho de volta
// que não fosse apagar o banco.
func sincronizarPapelApp(ctx context.Context, pool *db.Pool, senha string) error {
	if senha == "" {
		return nil // local sem variável: vale a senha do init
	}
	var existe bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'meirinho_app')`).Scan(&existe); err != nil {
		return err
	}
	// Papel não aceita parâmetro em DDL; format(%L) faz o escape do literal
	// no próprio Postgres.
	cmd := "ALTER ROLE meirinho_app WITH LOGIN PASSWORD %L"
	if !existe {
		cmd = "CREATE ROLE meirinho_app WITH LOGIN PASSWORD %L"
	}
	var sql string
	if err := pool.QueryRow(ctx, `SELECT format('`+cmd+`', $1::text)`, senha).Scan(&sql); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, sql)
	return err
}

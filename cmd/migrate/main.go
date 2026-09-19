// Binário de migration. Roda uma vez no boot do compose, antes dos serviços,
// conectado como dono do banco (os serviços conectam como meirinho_app).
package main

import (
	"context"
	"fmt"
	"net/url"
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

	dsn := config.MustString("DATABASE_URL")
	pool, err := conectar(ctx, dsn)
	if db.ErroDeAutenticacao(err) {
		// O volume do Postgres guarda a senha da PRIMEIRA subida. Se ele nasceu
		// antes das variáveis estarem definidas, a senha do dono é a padrão.
		// Nesse caso, entra com ela e troca para a senha configurada — deixar a
		// padrão seria pior, e não há outro caminho sem apagar o banco.
		log.Warn("senha do dono recusada; tentando a senha padrão do primeiro deploy")
		err = trocarSenhaPadraoDoDono(ctx, dsn)
		if err == nil {
			log.Info("senha do dono atualizada para a POSTGRES_PASSWORD configurada")
			pool, err = conectar(ctx, dsn)
		}
	}
	if err != nil {
		log.Error("conectar ao postgres; se for falha de autenticação, a POSTGRES_PASSWORD "+
			"não é a mesma com que o volume foi criado", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
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

// senhaPadraoDono é a POSTGRES_PASSWORD padrão do docker-compose.yml.
const senhaPadraoDono = "meirinho"

func conectar(ctx context.Context, dsn string) (*db.Pool, error) {
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Aguardar(ctx, 60); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// trocarSenhaPadraoDoDono entra com a senha padrão e define a senha que está
// no DSN configurado. Só tem efeito se o banco ainda estiver com a padrão.
func trocarSenhaPadraoDoDono(ctx context.Context, dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	nova, _ := u.User.Password()
	if nova == "" || nova == senhaPadraoDono {
		return fmt.Errorf("senha configurada é a própria padrão; nada a tentar")
	}
	u.User = url.UserPassword(u.User.Username(), senhaPadraoDono)
	pool, err := conectar(ctx, u.String())
	if err != nil {
		return fmt.Errorf("senha padrão também recusada: %w", err)
	}
	defer pool.Close()

	var sql string
	if err := pool.QueryRow(ctx, `SELECT format('ALTER ROLE %I WITH PASSWORD %L', $1::text, $2::text)`,
		u.User.Username(), nova).Scan(&sql); err != nil {
		return err
	}
	_, err = pool.Exec(ctx, sql)
	return err
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

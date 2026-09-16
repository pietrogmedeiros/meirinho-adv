// Binário de migration. Roda uma vez no boot do compose, antes dos serviços,
// conectado como dono do banco (os serviços conectam como meirinho_app).
package main

import (
	"context"
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
		log.Error("aguardar postgres", "err", err)
		panic(err)
	}
	if err := db.Migrar(ctx, pool, log); err != nil {
		log.Error("migrar", "err", err)
		panic(err)
	}
	log.Info("migrations em dia")
}

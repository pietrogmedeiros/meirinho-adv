// Package events é o barramento entre os serviços, sobre Redis Streams.
//
// Redis Streams em vez de Kafka/RabbitMQ: no volume desta fase ele entrega o
// que precisamos (consumer group, ack, replay) sem trazer um broker inteiro
// para operar. O contrato aqui (Publish/Consume sobre Envelope JSON) é neutro
// o bastante para trocar por Kafka depois sem mexer nos handlers.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	campoPayload = "payload"
	maxLen       = 10000 // trim aproximado: o stream não cresce sem limite
)

type Bus struct {
	rdb *redis.Client
	log *slog.Logger
}

func New(addr string, logger *slog.Logger) *Bus {
	return &Bus{rdb: redis.NewClient(&redis.Options{Addr: addr}), log: logger}
}

func (b *Bus) Aguardar(ctx context.Context, tentativas int) error {
	var last error
	for i := 0; i < tentativas; i++ {
		if err := b.rdb.Ping(ctx).Err(); err == nil {
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
	return fmt.Errorf("redis indisponível: %w", last)
}

func (b *Bus) Close() error { return b.rdb.Close() }

// Publish serializa o payload e o coloca no stream.
func (b *Bus) Publish(ctx context.Context, stream string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializar evento %s: %w", stream, err)
	}
	id, err := b.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: maxLen,
		Approx: true,
		Values: map[string]any{campoPayload: string(raw)},
	}).Result()
	if err != nil {
		return fmt.Errorf("publicar em %s: %w", stream, err)
	}
	b.log.Info("evento publicado", "stream", stream, "event_id", id)
	return nil
}

// Handler processa uma mensagem. Devolver erro faz a mensagem NÃO ser
// confirmada — ela fica pendente no grupo e pode ser reprocessada.
type Handler func(ctx context.Context, raw []byte) error

// Consume roda um consumer group até o contexto ser cancelado.
//
// Grupo por serviço (não por instância): assim várias réplicas do mesmo worker
// dividem a carga, enquanto serviços diferentes recebem cada um sua cópia do
// evento.
func (b *Bus) Consume(ctx context.Context, stream, grupo string, h Handler) error {
	consumidor := grupo + "-" + uuid.NewString()[:8]

	// MKSTREAM cria o stream se ainda não existir; BUSYGROUP é esperado quando
	// outra réplica já criou o grupo.
	if err := b.rdb.XGroupCreateMkStream(ctx, stream, grupo, "0").Err(); err != nil &&
		!errors.Is(err, redis.Nil) && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("criar grupo %s em %s: %w", grupo, stream, err)
	}

	log := b.log.With("stream", stream, "grupo", grupo, "consumidor", consumidor)
	log.Info("consumidor iniciado")

	for {
		select {
		case <-ctx.Done():
			log.Info("consumidor encerrado")
			return nil
		default:
		}

		res, err := b.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    grupo,
			Consumer: consumidor,
			Streams:  []string{stream, ">"},
			Count:    10,
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			log.Error("erro lendo stream", "err", err)
			time.Sleep(time.Second)
			continue
		}

		for _, s := range res {
			for _, msg := range s.Messages {
				payload, _ := msg.Values[campoPayload].(string)
				msgLog := log.With("event_id", msg.ID)

				if err := h(ctx, []byte(payload)); err != nil {
					// Sem ack: a mensagem fica pendente para reprocessamento.
					msgLog.Error("handler falhou, mensagem não confirmada", "err", err)
					continue
				}
				if err := b.rdb.XAck(ctx, stream, grupo, msg.ID).Err(); err != nil {
					msgLog.Error("falha no ack", "err", err)
				}
			}
		}
	}
}

// Decode é o par de Publish no lado do consumidor.
func Decode[T any](raw []byte) (T, error) {
	var v T
	err := json.Unmarshal(raw, &v)
	return v, err
}

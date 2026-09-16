package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/db"
)

var errNaoEncontrada = errors.New("notificação não encontrada")

type notificacao struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"-"`
	Tipo         domain.TipoNotificacao `json:"tipo"`
	Categoria    string                 `json:"categoria"`
	Titulo       string                 `json:"titulo"`
	Corpo        string                 `json:"corpo"`
	Urgencia     *domain.Urgencia       `json:"urgencia,omitempty"`
	ReferenciaID *string                `json:"referencia_id,omitempty"`
	Lida         bool                   `json:"lida"`
	CreatedAt    time.Time              `json:"created_at"`
}

type store struct{ pool *db.Pool }

func (s *store) criar(ctx context.Context, n notificacao) (*notificacao, error) {
	var out notificacao
	out.TenantID = n.TenantID
	err := s.pool.TenantTx(ctx, n.TenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO notification.notifications
			       (tenant_id, tipo, categoria, titulo, corpo, urgencia, referencia_id, enviado_em)
			VALUES ($1, $2, $3, $4, $5, $6, $7, now())
			RETURNING id, tipo, categoria, titulo, corpo, urgencia, referencia_id, lida, created_at`,
			n.TenantID, string(n.Tipo), n.Categoria, n.Titulo, n.Corpo, n.Urgencia, n.ReferenciaID,
		).Scan(&out.ID, &out.Tipo, &out.Categoria, &out.Titulo, &out.Corpo,
			&out.Urgencia, &out.ReferenciaID, &out.Lida, &out.CreatedAt)
	})
	if err != nil {
		return nil, fmt.Errorf("criar notificação: %w", err)
	}
	return &out, nil
}

// listar filtra por não lidas quando pedido — é o caso de uso principal da
// tela inicial, e o índice parcial notifications_tenant_nao_lidas_idx existe
// justamente para ele.
func (s *store) listar(ctx context.Context, tenantID string, apenasNaoLidas bool) ([]notificacao, error) {
	itens := []notificacao{}
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tipo, categoria, titulo, corpo, urgencia, referencia_id, lida, created_at
			  FROM notification.notifications
			 WHERE ($1 = FALSE OR NOT lida)
			 ORDER BY created_at DESC
			 LIMIT 200`, apenasNaoLidas)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var n notificacao
			if err := rows.Scan(&n.ID, &n.Tipo, &n.Categoria, &n.Titulo, &n.Corpo,
				&n.Urgencia, &n.ReferenciaID, &n.Lida, &n.CreatedAt); err != nil {
				return err
			}
			itens = append(itens, n)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("listar notificações: %w", err)
	}
	return itens, nil
}

func (s *store) marcarLida(ctx context.Context, tenantID, id string) error {
	return s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE notification.notifications SET lida = TRUE WHERE id = $1`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errNaoEncontrada
		}
		return nil
	})
}

func (s *store) marcarTodasLidas(ctx context.Context, tenantID string) (int64, error) {
	var n int64
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE notification.notifications SET lida = TRUE WHERE NOT lida`)
		if err != nil {
			return err
		}
		n = tag.RowsAffected()
		return nil
	})
	return n, err
}

func (s *store) naoLidas(ctx context.Context, tenantID string) (int, error) {
	var n int
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT count(*) FROM notification.notifications WHERE NOT lida`).Scan(&n)
	})
	return n, err
}

// marcarMovimentacaoNotificada fecha o ciclo da movimentação.
//
// A coluna vive no schema process, de outro serviço. Escrever aqui é uma
// concessão consciente: o notification-service é o único que sabe o instante
// em que o advogado foi efetivamente avisado, e duplicar isso numa tabela
// própria só para respeitar a fronteira criaria duas verdades sobre o mesmo
// fato.
func (s *store) marcarMovimentacaoNotificada(ctx context.Context, tenantID, movementID string) error {
	return s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE process.movements SET notificado_em = now() WHERE id = $1`, movementID)
		return err
	})
}

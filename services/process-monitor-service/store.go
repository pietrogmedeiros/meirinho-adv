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

var (
	errNaoEncontrado = errors.New("processo não encontrado")
	errDuplicado     = errors.New("processo já monitorado")
)

type processo struct {
	ID            string               `json:"id"`
	NumeroCNJ     string               `json:"numero_cnj"`
	Titulo        string               `json:"titulo"`
	Tribunal      string               `json:"tribunal"`
	AreaDoDireito domain.AreaDoDireito `json:"area_do_direito"`
	Provider      string               `json:"provider"`
	Status        string               `json:"status"`
	CreatedAt     time.Time            `json:"created_at"`
}

type movimentacao struct {
	ID               string           `json:"id"`
	ProcessID        string           `json:"process_id"`
	Descricao        string           `json:"descricao"`
	Data             time.Time        `json:"data"`
	Urgencia         *domain.Urgencia `json:"urgencia,omitempty"`
	Sugestao         *string          `json:"sugestao,omitempty"`
	PrazoDias        *int             `json:"prazo_dias,omitempty"`
	FallbackAplicado bool             `json:"fallback_aplicado"`
	CreatedAt        time.Time        `json:"created_at"`
}

type store struct{ pool *db.Pool }

func (s *store) criar(ctx context.Context, tenantID, cnj, titulo, tribunal string, area domain.AreaDoDireito, provider string) (*processo, error) {
	var p processo
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO process.monitored_processes
			       (tenant_id, numero_cnj, titulo, tribunal, area_do_direito, provider)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, numero_cnj, titulo, tribunal, area_do_direito, provider, status, created_at`,
			tenantID, cnj, titulo, tribunal, string(area), provider,
		).Scan(&p.ID, &p.NumeroCNJ, &p.Titulo, &p.Tribunal, &p.AreaDoDireito, &p.Provider, &p.Status, &p.CreatedAt)
	})
	if err != nil {
		// 23505 = unique_violation: o índice (tenant_id, numero_cnj) já barrou.
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return nil, errDuplicado
		}
		return nil, fmt.Errorf("criar processo: %w", err)
	}
	return &p, nil
}

func (s *store) listar(ctx context.Context, tenantID string) ([]processo, error) {
	itens := []processo{}
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, numero_cnj, titulo, tribunal, area_do_direito, provider, status, created_at
			  FROM process.monitored_processes
			 ORDER BY created_at DESC
			 LIMIT 500`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p processo
			if err := rows.Scan(&p.ID, &p.NumeroCNJ, &p.Titulo, &p.Tribunal,
				&p.AreaDoDireito, &p.Provider, &p.Status, &p.CreatedAt); err != nil {
				return err
			}
			itens = append(itens, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("listar processos: %w", err)
	}
	return itens, nil
}

// arquivar em vez de deletar: a movimentação já recebida continua valendo como
// histórico, e o advogado pode ter agido com base nela.
func (s *store) arquivar(ctx context.Context, tenantID, id string) error {
	return s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE process.monitored_processes SET status = 'arquivado' WHERE id = $1`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errNaoEncontrado
		}
		return nil
	})
}

func (s *store) movimentacoes(ctx context.Context, tenantID, processID string) ([]movimentacao, error) {
	itens := []movimentacao{}
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, process_id, descricao, data, urgencia_classificada,
			       sugestao, prazo_dias, fallback_aplicado, created_at
			  FROM process.movements
			 WHERE process_id = $1
			 ORDER BY data DESC
			 LIMIT 500`, processID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m movimentacao
			if err := rows.Scan(&m.ID, &m.ProcessID, &m.Descricao, &m.Data, &m.Urgencia,
				&m.Sugestao, &m.PrazoDias, &m.FallbackAplicado, &m.CreatedAt); err != nil {
				return err
			}
			itens = append(itens, m)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("listar movimentações: %w", err)
	}
	return itens, nil
}

// tenants roda fora do escopo de tenant: o poller precisa da lista completa
// para depois abrir uma transação escopada por cada um. auth.tenants não tem
// RLS justamente por ser consultada antes de haver tenant corrente.
func (s *store) tenants(ctx context.Context) ([]string, error) {
	var ids []string
	err := s.pool.AdminTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id FROM auth.tenants`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("listar tenants: %w", err)
	}
	return ids, nil
}

type processoAtivo struct {
	ID          string
	NumeroCNJ   string
	ProviderRef string
	Area        domain.AreaDoDireito
}

func (s *store) ativosDoTenant(ctx context.Context, tenantID string) ([]processoAtivo, error) {
	var itens []processoAtivo
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, numero_cnj, provider_ref, area_do_direito
			  FROM process.monitored_processes
			 WHERE status = 'ativo'`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p processoAtivo
			if err := rows.Scan(&p.ID, &p.NumeroCNJ, &p.ProviderRef, &p.Area); err != nil {
				return err
			}
			itens = append(itens, p)
		}
		return rows.Err()
	})
	return itens, err
}

// inserirMovimentacao devolve id vazio quando o andamento já existia. O
// ON CONFLICT DO NOTHING é o que sustenta a idempotência: o provider reenvia,
// o banco ignora, e nenhum evento duplicado entra no barramento.
func (s *store) inserirMovimentacao(ctx context.Context, tenantID, processID string, mv MovimentacaoProvider) (string, error) {
	var id string
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO process.movements
			       (tenant_id, process_id, provider_event_id, descricao, data)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (process_id, provider_event_id) DO NOTHING
			RETURNING id`,
			tenantID, processID, mv.EventID, mv.Descricao, mv.Data,
		).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // já existia
		}
		return err
	})
	if err != nil {
		return "", fmt.Errorf("inserir movimentação: %w", err)
	}
	return id, nil
}

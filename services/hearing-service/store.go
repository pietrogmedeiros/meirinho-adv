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

var errNaoEncontrada = errors.New("audiência não encontrada")

type audiencia struct {
	ID            string               `json:"id"`
	Titulo        string               `json:"titulo"`
	NomeArquivo   string               `json:"nome_arquivo"`
	AreaDoDireito domain.AreaDoDireito `json:"area_do_direito"`
	Status        domain.HearingStatus `json:"status"`
	Transcript    *string              `json:"transcript,omitempty"`
	Summary       *string              `json:"summary,omitempty"`
	Suggestion    *string              `json:"suggestion,omitempty"`
	Erro          *string              `json:"erro,omitempty"`
	AudioURL      string               `json:"audio_url,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type store struct{ pool *db.Pool }

func (s *store) criar(ctx context.Context, tenantID, titulo, objectKey, nomeArquivo string, area domain.AreaDoDireito) (*audiencia, error) {
	var a audiencia
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO hearing.hearings (tenant_id, titulo, object_key, nome_arquivo, area_do_direito, status)
			VALUES ($1, $2, $3, $4, $5, 'uploaded')
			RETURNING id, titulo, nome_arquivo, area_do_direito, status, created_at, updated_at`,
			tenantID, titulo, objectKey, nomeArquivo, string(area),
		).Scan(&a.ID, &a.Titulo, &a.NomeArquivo, &a.AreaDoDireito, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	})
	if err != nil {
		return nil, fmt.Errorf("criar audiência: %w", err)
	}
	return &a, nil
}

// listar devolve o resumo da audiência sem a transcrição inteira — a listagem
// não precisa carregar megabytes de texto.
func (s *store) listar(ctx context.Context, tenantID string) ([]audiencia, error) {
	itens := []audiencia{}
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, titulo, nome_arquivo, area_do_direito, status, summary, erro, created_at, updated_at
			  FROM hearing.hearings
			 ORDER BY created_at DESC
			 LIMIT 200`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a audiencia
			if err := rows.Scan(&a.ID, &a.Titulo, &a.NomeArquivo, &a.AreaDoDireito,
				&a.Status, &a.Summary, &a.Erro, &a.CreatedAt, &a.UpdatedAt); err != nil {
				return err
			}
			itens = append(itens, a)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("listar audiências: %w", err)
	}
	return itens, nil
}

func (s *store) buscar(ctx context.Context, tenantID, id string) (*audiencia, string, error) {
	var a audiencia
	var objectKey string
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, titulo, nome_arquivo, area_do_direito, status,
			       transcript, summary, suggestion, erro, object_key, created_at, updated_at
			  FROM hearing.hearings WHERE id = $1`, id,
		).Scan(&a.ID, &a.Titulo, &a.NomeArquivo, &a.AreaDoDireito, &a.Status,
			&a.Transcript, &a.Summary, &a.Suggestion, &a.Erro, &objectKey, &a.CreatedAt, &a.UpdatedAt)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", errNaoEncontrada
	}
	if err != nil {
		return nil, "", fmt.Errorf("buscar audiência: %w", err)
	}
	return &a, objectKey, nil
}

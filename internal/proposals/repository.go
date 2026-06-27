package proposals

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type similarResult struct {
	ID         string
	Similarity float64
}

func (r *Repository) FindSimilar(ctx context.Context, embedding []float32, threshold float64) (*similarResult, error) {
	vec := pgvector.NewVector(embedding)
	row := r.db.QueryRow(ctx, `
		SELECT id, 1 - (embedding <=> $1::vector) AS similarity
		FROM proposals
		WHERE 1 - (embedding <=> $1::vector) >= $2
		ORDER BY similarity DESC
		LIMIT 1
	`, vec, threshold)

	var res similarResult
	err := row.Scan(&res.ID, &res.Similarity)
	if err != nil {
		// pgx returns pgx.ErrNoRows when nothing found
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &res, nil
}

func (r *Repository) Insert(ctx context.Context, req SubmitRequest, embedding []float32, scorecard Scorecard) (*Proposal, error) {
	scorecardJSON, err := json.Marshal(scorecard)
	if err != nil {
		return nil, fmt.Errorf("marshal scorecard: %w", err)
	}

	vec := pgvector.NewVector(embedding)

	var p Proposal
	err = r.db.QueryRow(ctx, `
		INSERT INTO proposals (title, description, data_sources, intended_use, embedding, risk_score, scorecard)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, description, data_sources, intended_use, risk_score, scorecard, duplicate_of, created_at
	`, req.Title, req.Description, req.DataSources, req.IntendedUse, vec, scorecard.OverallRisk, scorecardJSON,
	).Scan(&p.ID, &p.Title, &p.Description, &p.DataSources, &p.IntendedUse, &p.RiskScore, &p.Scorecard, &p.DuplicateOf, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}

	return &p, nil
}

func (r *Repository) InsertDuplicate(ctx context.Context, req SubmitRequest, embedding []float32, duplicateOf string) (string, error) {
	vec := pgvector.NewVector(embedding)
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO proposals (title, description, data_sources, intended_use, embedding, risk_score, duplicate_of)
		VALUES ($1, $2, $3, $4, $5, 'duplicate', $6)
		RETURNING id
	`, req.Title, req.Description, req.DataSources, req.IntendedUse, vec, duplicateOf).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert duplicate: %w", err)
	}
	return id, nil
}

func (r *Repository) List(ctx context.Context) ([]Proposal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, data_sources, intended_use, risk_score, scorecard, duplicate_of, created_at
		FROM proposals
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var proposals []Proposal
	for rows.Next() {
		var p Proposal
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.DataSources, &p.IntendedUse, &p.RiskScore, &p.Scorecard, &p.DuplicateOf, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		proposals = append(proposals, p)
	}
	return proposals, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Proposal, error) {
	var p Proposal
	err := r.db.QueryRow(ctx, `
		SELECT id, title, description, data_sources, intended_use, risk_score, scorecard, duplicate_of, created_at
		FROM proposals
		WHERE id = $1
	`, id).Scan(&p.ID, &p.Title, &p.Description, &p.DataSources, &p.IntendedUse, &p.RiskScore, &p.Scorecard, &p.DuplicateOf, &p.CreatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &p, nil
}

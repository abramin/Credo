package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"credo/internal/evidence/vc/models"
	vcsq "credo/internal/evidence/vc/store/sqlc"
	id "credo/pkg/domain"
	"credo/pkg/platform/sentinel"

	"github.com/google/uuid"
)

// PostgresStore persists credentials in PostgreSQL.
type PostgresStore struct {
	db      *sql.DB
	queries *vcsq.Queries
}

// NewPostgres constructs a PostgreSQL-backed credential store.
func NewPostgres(db *sql.DB) *PostgresStore {
	return &PostgresStore{
		db:      db,
		queries: vcsq.New(db),
	}
}

func (s *PostgresStore) Save(ctx context.Context, credential models.CredentialRecord) error {
	claimsBytes, err := json.Marshal(credential.Claims)
	if err != nil {
		return fmt.Errorf("marshal credential claims: %w", err)
	}
	isOver18, verifiedVia := extractAgeOver18Claims(credential)
	err = s.queries.UpsertCredential(ctx, vcsq.UpsertCredentialParams{
		ID:          credential.ID.String(),
		Type:        string(credential.Type),
		SubjectID:   uuid.UUID(credential.Subject),
		Issuer:      credential.Issuer,
		IssuedAt:    credential.IssuedAt,
		Claims:      claimsBytes,
		IsOver18:    isOver18,
		VerifiedVia: verifiedVia,
	})
	if err != nil {
		return fmt.Errorf("save credential: %w", err)
	}
	return nil
}

func (s *PostgresStore) FindByID(ctx context.Context, credentialID models.CredentialID) (models.CredentialRecord, error) {
	row, err := s.queries.GetCredentialByID(ctx, credentialID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.CredentialRecord{}, sentinel.ErrNotFound
		}
		return models.CredentialRecord{}, fmt.Errorf("find credential by id: %w", err)
	}
	record, err := toCredentialRecord(credentialRow{
		ID:          row.ID,
		Type:        row.Type,
		SubjectID:   row.SubjectID,
		Issuer:      row.Issuer,
		IssuedAt:    row.IssuedAt,
		Claims:      row.Claims,
		IsOver18:    row.IsOver18,
		VerifiedVia: row.VerifiedVia,
	})
	if err != nil {
		return models.CredentialRecord{}, err
	}
	return record, nil
}

func (s *PostgresStore) FindBySubjectAndType(ctx context.Context, subject id.UserID, credType models.CredentialType) (models.CredentialRecord, error) {
	row, err := s.queries.GetCredentialBySubjectAndType(ctx, vcsq.GetCredentialBySubjectAndTypeParams{
		SubjectID: uuid.UUID(subject),
		Type:      string(credType),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.CredentialRecord{}, sentinel.ErrNotFound
		}
		return models.CredentialRecord{}, fmt.Errorf("find credential by subject and type: %w", err)
	}
	record, err := toCredentialRecord(credentialRow{
		ID:          row.ID,
		Type:        row.Type,
		SubjectID:   row.SubjectID,
		Issuer:      row.Issuer,
		IssuedAt:    row.IssuedAt,
		Claims:      row.Claims,
		IsOver18:    row.IsOver18,
		VerifiedVia: row.VerifiedVia,
	})
	if err != nil {
		return models.CredentialRecord{}, err
	}
	return record, nil
}

type credentialRow struct {
	ID          string
	Type        string
	SubjectID   uuid.UUID
	Issuer      string
	IssuedAt    time.Time
	Claims      interface{}
	IsOver18    sql.NullBool
	VerifiedVia sql.NullString
}

func toCredentialRecord(row credentialRow) (models.CredentialRecord, error) {
	var record models.CredentialRecord
	record.ID = models.CredentialID(row.ID)
	record.Type = models.CredentialType(row.Type)
	record.Subject = id.UserID(row.SubjectID)
	record.Issuer = row.Issuer
	record.IssuedAt = row.IssuedAt

	claimsBytes, err := parseClaims(row.Claims)
	if err != nil {
		return models.CredentialRecord{}, err
	}

	var claims models.Claims
	if len(claimsBytes) > 0 {
		if err := json.Unmarshal(claimsBytes, &claims); err != nil {
			return models.CredentialRecord{}, fmt.Errorf("unmarshal credential claims: %w", err)
		}
	}
	if claims == nil {
		claims = models.Claims{}
	}
	if record.Type == models.CredentialTypeAgeOver18 {
		if row.IsOver18.Valid {
			claims["is_over_18"] = row.IsOver18.Bool
		}
		if row.VerifiedVia.Valid && row.VerifiedVia.String != "" {
			claims["verified_via"] = row.VerifiedVia.String
		}
	}
	record.Claims = claims
	return record, nil
}

func parseClaims(raw interface{}) ([]byte, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	case json.RawMessage:
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("unexpected claims type: %T", raw)
	}
}

func extractAgeOver18Claims(credential models.CredentialRecord) (sql.NullBool, sql.NullString) {
	if credential.Type != models.CredentialTypeAgeOver18 {
		return sql.NullBool{}, sql.NullString{}
	}

	var isOver18 sql.NullBool
	var verifiedVia sql.NullString
	if value, ok := credential.Claims["is_over_18"].(bool); ok {
		isOver18 = sql.NullBool{Bool: value, Valid: true}
	}
	if value, ok := credential.Claims["verified_via"].(string); ok && value != "" {
		verifiedVia = sql.NullString{String: value, Valid: true}
	}
	return isOver18, verifiedVia
}

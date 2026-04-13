package sample

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"evaluating_platform/internal/crypto"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/storage"
)

// ComposedAttackManager manages ready-to-run combined attack datasets.
type ComposedAttackManager struct {
	repo  *repository.ComposedAttackRepository
	store *storage.MinIOClient
}

func NewComposedAttackManager(
	repo *repository.ComposedAttackRepository,
	store *storage.MinIOClient,
) *ComposedAttackManager {
	return &ComposedAttackManager{repo: repo, store: store}
}

func (m *ComposedAttackManager) Upload(ctx context.Context, expertID uuid.UUID, subType, name, description string, csvData []byte) (*model.ComposedAttack, error) {
	normalizedCSV, err := NormalizeCSVData(csvData)
	if err != nil {
		return nil, err
	}

	rows, err := ParseTwoColumnCSV(normalizedCSV)
	if err != nil {
		return nil, err
	}

	itemID := uuid.New()
	storagePath := fmt.Sprintf("composed_attacks/%s/%s.csv", expertID.String(), itemID.String())
	if err := m.store.Upload(ctx, storagePath, normalizedCSV, "text/csv; charset=utf-8"); err != nil {
		return nil, fmt.Errorf("upload CSV: %w", err)
	}

	item := &model.ComposedAttack{
		ID:          itemID,
		ExpertID:    expertID,
		SubType:     subType,
		Name:        name,
		Description: description,
		StoragePath: storagePath,
		FileHash:    crypto.SHA256Hex(normalizedCSV),
		SampleCount: len(rows),
		FileSize:    int64(len(normalizedCSV)),
		Status:      "published",
		Visibility:  "public",
	}
	if err := m.repo.Create(ctx, item); err != nil {
		_ = m.store.Delete(ctx, storagePath)
		return nil, fmt.Errorf("save composed attack record: %w", err)
	}

	return item, nil
}

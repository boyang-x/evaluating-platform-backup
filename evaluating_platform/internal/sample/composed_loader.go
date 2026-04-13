package sample

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/storage"
)

// ComposedAttackLoader loads ready-to-run composed attacks from storage.
type ComposedAttackLoader struct {
	repo  *repository.ComposedAttackRepository
	store *storage.MinIOClient
}

func NewComposedAttackLoader(
	repo *repository.ComposedAttackRepository,
	store *storage.MinIOClient,
) *ComposedAttackLoader {
	return &ComposedAttackLoader{repo: repo, store: store}
}

type LoadedComposedAttacks struct {
	Payloads []model.AttackPayload
}

func (lc *LoadedComposedAttacks) Close() {
	lc.Payloads = nil
}

func (l *ComposedAttackLoader) Load(ctx context.Context, composedAttackID uuid.UUID) (*LoadedComposedAttacks, error) {
	item, err := l.repo.GetByID(ctx, composedAttackID)
	if err != nil {
		return nil, fmt.Errorf("get composed attack: %w", err)
	}

	csvData, err := l.store.Download(ctx, item.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("download csv: %w", err)
	}

	payloads, err := parsePayloads(csvData)
	if err != nil {
		return nil, fmt.Errorf("parse payloads: %w", err)
	}

	return &LoadedComposedAttacks{Payloads: payloads}, nil
}

func (l *ComposedAttackLoader) Preview(ctx context.Context, composedAttackID uuid.UUID, n int) ([]model.AttackPayload, error) {
	loaded, err := l.Load(ctx, composedAttackID)
	if err != nil {
		return nil, err
	}
	defer loaded.Close()

	if n > len(loaded.Payloads) {
		n = len(loaded.Payloads)
	}
	preview := make([]model.AttackPayload, n)
	copy(preview, loaded.Payloads[:n])
	return preview, nil
}

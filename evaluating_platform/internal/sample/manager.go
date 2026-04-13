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

// Manager 攻击样本管理器（上传明文、加载明文）
type Manager struct {
	sampleRepo *repository.AttackSampleRepository
	store      *storage.MinIOClient
}

// NewManager 创建样本管理器
func NewManager(
	sampleRepo *repository.AttackSampleRepository,
	store *storage.MinIOClient,
) *Manager {
	return &Manager{sampleRepo: sampleRepo, store: store}
}

// Upload 上传 CSV 样本：解析 → 计算哈希 → 存 MinIO → 写 DB
func (m *Manager) Upload(ctx context.Context, expertID uuid.UUID, subType, name, description string, csvData []byte) (*model.AttackSample, error) {
	normalizedCSV, err := NormalizeCSVData(csvData)
	if err != nil {
		return nil, err
	}

	// 1. 解析 CSV 校验格式（两列：序号+数据，无表头）
	rows, err := ParseTwoColumnCSV(normalizedCSV)
	if err != nil {
		return nil, err
	}

	sampleCount := len(rows)

	// 2. 计算原始文件 SHA-256
	fileHash := crypto.SHA256Hex(normalizedCSV)

	// 3. 上传明文 CSV 到 MinIO
	sampleID := uuid.New()
	storagePath := fmt.Sprintf("samples/%s/%s.csv", expertID.String(), sampleID.String())
	if err := m.store.Upload(ctx, storagePath, normalizedCSV, "text/csv; charset=utf-8"); err != nil {
		return nil, fmt.Errorf("upload CSV: %w", err)
	}

	// 4. 写入 DB
	sample := &model.AttackSample{
		ID:          sampleID,
		ExpertID:    expertID,
		SubType:     subType,
		Name:        name,
		Description: description,
		StoragePath: storagePath,
		FileHash:    fileHash,
		SampleCount: sampleCount,
		FileSize:    int64(len(normalizedCSV)),
		Status:      "published",
		Visibility:  "public",
	}
	if err := m.sampleRepo.Create(ctx, sample); err != nil {
		// 回滚 MinIO
		_ = m.store.Delete(ctx, storagePath)
		return nil, fmt.Errorf("save sample record: %w", err)
	}

	return sample, nil
}

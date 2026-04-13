package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
)

type PayloadItem struct {
	Index            int    `json:"index"`
	OriginalQuestion string `json:"original_question"`
	FinalPayloadText string `json:"final_payload_text"`
	StrategySummary  string `json:"strategy_summary,omitempty"`
}

type PayloadDataset struct {
	DatasetID    string        `json:"dataset_id"`
	ResourceType string        `json:"resource_type"`
	Source       string        `json:"source"`
	Model        string        `json:"model"`
	CreatedAt    time.Time     `json:"created_at"`
	Items        []PayloadItem `json:"items"`
}

type DatasetStore struct {
	baseDir    string
	datasetDir string
	exportDir  string
}

func NewDatasetStore(baseDir string) (*DatasetStore, error) {
	datasetDir := filepath.Join(baseDir, "datasets")
	exportDir := filepath.Join(baseDir, "exports")
	for _, dir := range []string{baseDir, datasetDir, exportDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create artifact dir %s: %w", dir, err)
		}
	}
	return &DatasetStore{
		baseDir:    baseDir,
		datasetDir: datasetDir,
		exportDir:  exportDir,
	}, nil
}

func (s *DatasetStore) Save(dataset PayloadDataset) error {
	path := s.datasetPath(dataset.DatasetID)
	bytes, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dataset: %w", err)
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fmt.Errorf("write dataset: %w", err)
	}
	return nil
}

func (s *DatasetStore) Load(datasetID string) (*PayloadDataset, error) {
	bytes, err := os.ReadFile(s.datasetPath(datasetID))
	if err != nil {
		return nil, fmt.Errorf("read dataset: %w", err)
	}
	var dataset PayloadDataset
	if err := json.Unmarshal(bytes, &dataset); err != nil {
		return nil, fmt.Errorf("parse dataset: %w", err)
	}
	sort.Slice(dataset.Items, func(i, j int) bool {
		return dataset.Items[i].Index < dataset.Items[j].Index
	})
	return &dataset, nil
}

func (s *DatasetStore) ExportCSV(datasetID, outputName string) (string, int, error) {
	dataset, err := s.Load(datasetID)
	if err != nil {
		return "", 0, err
	}
	if outputName == "" {
		outputName = fmt.Sprintf("ccbos_payloads_%s.csv", datasetID)
	}
	path := filepath.Join(s.exportDir, outputName)
	file, err := os.Create(path)
	if err != nil {
		return "", 0, fmt.Errorf("create csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"index", "original_question", "final_payload_text", "strategy_summary"}); err != nil {
		return "", 0, fmt.Errorf("write csv header: %w", err)
	}
	for _, item := range dataset.Items {
		if err := writer.Write([]string{
			fmt.Sprintf("%d", item.Index),
			item.OriginalQuestion,
			item.FinalPayloadText,
			item.StrategySummary,
		}); err != nil {
			return "", 0, fmt.Errorf("write csv row: %w", err)
		}
	}
	return path, len(dataset.Items), nil
}

func (s *DatasetStore) datasetPath(datasetID string) string {
	return filepath.Join(s.datasetDir, fmt.Sprintf("%s.json", datasetID))
}

func newDatasetID() string {
	return uuid.NewString()
}
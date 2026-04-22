package skill

import "encoding/json"

type RuntimeRequest struct {
	RunType        string            `json:"run_type"`
	ArchiveBase64  string            `json:"archive_base64"`
	Runtime        string            `json:"runtime"`
	Entrypoint     string            `json:"entrypoint"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	Input          json.RawMessage   `json:"input"`
	Env            map[string]string `json:"env,omitempty"`
}

type RuntimeResponse struct {
	Status           string          `json:"status"`
	ExitCode         *int            `json:"exit_code,omitempty"`
	Stdout           string          `json:"stdout"`
	Stderr           string          `json:"stderr"`
	Result           json.RawMessage `json:"result"`
	ValidationReport json.RawMessage `json:"validation_report"`
	Error            string          `json:"error"`
}

type GenerateInput struct {
	AssessmentID    string                 `json:"assessment_id,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	Goal            string                 `json:"goal,omitempty"`
	AssessmentTypes []string               `json:"assessment_types,omitempty"`
	RequestedCount  int                    `json:"requested_count,omitempty"`
	SourceSampleID  string                 `json:"source_sample_id,omitempty"`
	SourceSamples   []SourceSampleItem     `json:"source_samples,omitempty"`
	TargetProfile   map[string]interface{} `json:"target_profile,omitempty"`
	ExecutionMeta   map[string]interface{} `json:"execution_meta,omitempty"`
}

type SourceSampleItem struct {
	Index int    `json:"index,omitempty"`
	Text  string `json:"text"`
}

type GenerateRequest struct {
	AssessmentID    string
	UserID          string
	Goal            string
	AssessmentTypes []string
	RequestedCount  int
	SourceSampleID  string
	SourceSamples   []SourceSampleItem
}

type TargetProfile struct {
	Type     string `json:"type"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key,omitempty"`
	Model    string `json:"model,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

type GenerateResult struct {
	PayloadDataset PayloadDataset         `json:"payload_dataset"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type PayloadDataset struct {
	DatasetID string        `json:"dataset_id,omitempty"`
	Summary   string        `json:"summary,omitempty"`
	Count     int           `json:"count,omitempty"`
	Payloads  []PayloadItem `json:"payloads"`
}

type PayloadItem struct {
	ID               string `json:"id,omitempty"`
	OriginalQuestion string `json:"original_question,omitempty"`
	PayloadText      string `json:"payload_text"`
	QuestionSummary  string `json:"question_summary,omitempty"`
	PayloadSummary   string `json:"payload_summary,omitempty"`
	Language         string `json:"language,omitempty"`
	Sensitive        bool   `json:"sensitive,omitempty"`
}

package maclaw

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
)

type fakePlatformPayloadSampleStore struct {
	item model.AttackSample
}

func (s fakePlatformPayloadSampleStore) Preview(context.Context, uuid.UUID, int) ([]model.AttackPayload, error) {
	return []model.AttackPayload{{Index: 1, Data: "private sample payload"}}, nil
}

func (s fakePlatformPayloadSampleStore) GetByID(context.Context, uuid.UUID) (*model.AttackSample, error) {
	return &s.item, nil
}

type fakePlatformPayloadTemplateStore struct {
	item model.Template
}

func (s fakePlatformPayloadTemplateStore) GetByID(context.Context, uuid.UUID) (*model.Template, error) {
	return &s.item, nil
}

type fakePlatformPayloadComposedStore struct {
	item model.ComposedAttack
}

func (s fakePlatformPayloadComposedStore) Preview(context.Context, uuid.UUID, int) ([]model.AttackPayload, error) {
	return []model.AttackPayload{{Index: 1, Data: "private composed payload"}}, nil
}

func (s fakePlatformPayloadComposedStore) GetByID(context.Context, uuid.UUID) (*model.ComposedAttack, error) {
	return &s.item, nil
}

func TestPlatformRedteamPayloadProviderRejectsUnpublishedRefs(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	sampleID := uuid.New()
	templateID := uuid.New()
	composedID := uuid.New()
	provider := NewPlatformRedteamPayloadProvider(
		fakePlatformPayloadSampleStore{item: model.AttackSample{ID: sampleID, Status: "draft", Visibility: "private", UpdatedAt: now}},
		fakePlatformPayloadTemplateStore{item: model.Template{ID: templateID, Status: "draft", Visibility: "private", Content: "{{sample}}", UpdatedAt: now}},
		fakePlatformPayloadComposedStore{item: model.ComposedAttack{ID: composedID, Status: "draft", Visibility: "private", UpdatedAt: now}},
	)

	if _, err := provider.LoadSamplePayloads(context.Background(), "sample:"+sampleID.String(), 1); err == nil || !strings.Contains(err.Error(), "not published") {
		t.Fatalf("LoadSamplePayloads err = %v, want not published", err)
	}
	if _, err := provider.GetTemplate(context.Background(), "template:"+templateID.String()); err == nil || !strings.Contains(err.Error(), "not published") {
		t.Fatalf("GetTemplate err = %v, want not published", err)
	}
	if _, err := provider.LoadComposedPayloads(context.Background(), "composed_attack:"+composedID.String(), 1); err == nil || !strings.Contains(err.Error(), "not published") {
		t.Fatalf("LoadComposedPayloads err = %v, want not published", err)
	}
}

func TestPlatformRedteamPayloadProviderAllowsPublishedRefs(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	sampleID := uuid.New()
	templateID := uuid.New()
	composedID := uuid.New()
	provider := NewPlatformRedteamPayloadProvider(
		fakePlatformPayloadSampleStore{item: model.AttackSample{ID: sampleID, Status: "published", Visibility: "public", UpdatedAt: now}},
		fakePlatformPayloadTemplateStore{item: model.Template{ID: templateID, Status: "published", Visibility: "public", Content: "{{sample}}", UpdatedAt: now}},
		fakePlatformPayloadComposedStore{item: model.ComposedAttack{ID: composedID, Status: "published", Visibility: "public", UpdatedAt: now}},
	)

	if payloads, err := provider.LoadSamplePayloads(context.Background(), "sample:"+sampleID.String(), 1); err != nil || len(payloads) != 1 {
		t.Fatalf("LoadSamplePayloads payloads=%#v err=%v", payloads, err)
	}
	if tpl, err := provider.GetTemplate(context.Background(), "template:"+templateID.String()); err != nil || tpl == nil {
		t.Fatalf("GetTemplate tpl=%#v err=%v", tpl, err)
	}
	if payloads, err := provider.LoadComposedPayloads(context.Background(), "composed_attack:"+composedID.String(), 1); err != nil || len(payloads) != 1 {
		t.Fatalf("LoadComposedPayloads payloads=%#v err=%v", payloads, err)
	}
}

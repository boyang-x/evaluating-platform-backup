package maclaw

import "strings"

type RuntimeIdentity struct {
	UserID string
	Role   string
}

type InstanceMapping struct {
	UserID     string `json:"user_id,omitempty" mapstructure:"user_id"`
	Role       string `json:"role,omitempty" mapstructure:"role"`
	InstanceID string `json:"instance_id" mapstructure:"instance_id"`
}

type InstanceResolver struct {
	defaultInstanceID string
	mappings          []InstanceMapping
}

func NewInstanceResolver(defaultInstanceID string, mappings []InstanceMapping) *InstanceResolver {
	clean := make([]InstanceMapping, 0, len(mappings))
	for _, item := range mappings {
		item.UserID = strings.TrimSpace(item.UserID)
		item.Role = strings.TrimSpace(item.Role)
		item.InstanceID = strings.TrimSpace(item.InstanceID)
		if item.InstanceID == "" {
			continue
		}
		clean = append(clean, item)
	}
	return &InstanceResolver{
		defaultInstanceID: strings.TrimSpace(defaultInstanceID),
		mappings:          clean,
	}
}

func (r *InstanceResolver) Resolve(identity RuntimeIdentity) (string, bool) {
	if r == nil {
		return "", false
	}
	userID := strings.TrimSpace(identity.UserID)
	role := strings.TrimSpace(identity.Role)
	for _, item := range r.mappings {
		if item.UserID != "" && item.UserID == userID {
			return item.InstanceID, true
		}
	}
	for _, item := range r.mappings {
		if item.Role != "" && item.Role == role {
			return item.InstanceID, true
		}
	}
	if r.defaultInstanceID == "" {
		return "", false
	}
	return r.defaultInstanceID, true
}

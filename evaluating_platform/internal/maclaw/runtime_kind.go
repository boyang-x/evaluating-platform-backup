package maclaw

import (
	"fmt"
	"strings"
)

const (
	RuntimeKindMaclawSrv = "maclawsrv"
)

func NormalizeRuntimeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case RuntimeKindMaclawSrv, "maclaw-srv", "maclaw_srv":
		return RuntimeKindMaclawSrv
	case "":
		return RuntimeKindMaclawSrv
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

func NewRuntimeClient(kind string, cfg Config) (GatewayClient, error) {
	normalized := NormalizeRuntimeKind(kind)
	if normalized != RuntimeKindMaclawSrv {
		return nil, fmt.Errorf("unsupported maclaw runtime kind %q: only %q is supported", strings.TrimSpace(kind), RuntimeKindMaclawSrv)
	}
	return NewMaclawSrvClient(cfg)
}

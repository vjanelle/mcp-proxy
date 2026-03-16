package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
)

func requestID(payload []byte) (string, bool, error) {
	trimmed := strings.TrimSpace(string(payload))
	if strings.HasPrefix(trimmed, "[") {
		return "", false, fmt.Errorf("batch json-rpc requests are not supported")
	}

	var msg map[string]json.RawMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return "", false, fmt.Errorf("parse json-rpc message: %w", err)
	}

	idRaw, ok := msg["id"]
	if !ok {
		return "", false, nil
	}

	return normalizeJSON(idRaw), true, nil
}

func normalizeJSON(raw json.RawMessage) string {
	return strings.TrimSpace(string(raw))
}

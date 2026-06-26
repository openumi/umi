package umi

import (
	"context"
	"encoding/json"
	"fmt"
)

type Config struct {
	AppsRaw ModuleMap `json:"apps,omitempty"`
	apps    map[string]App

	cancelFunc context.CancelCauseFunc
}

func LoadAppsConfig(data []byte) (*Config, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	raw, ok := root["apps"]
	if !ok {
		return nil, fmt.Errorf(`missing required field "apps"`)
	}

	var appsRaw ModuleMap
	if err := json.Unmarshal(raw, &appsRaw); err != nil {
		return nil, fmt.Errorf(`decoding "apps": %w`, err)
	}

	return &Config{
		AppsRaw: appsRaw,
	}, nil
}

package editbrush

import (
	"encoding/json"
	"fmt"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/we/keys"
	"github.com/df-mc/we/service"
)

// BindBrush serialises cfg onto the item stack.
func BindBrush(i item.Stack, cfg service.BrushConfig) (item.Stack, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return item.Stack{}, err
	}
	name := fmt.Sprintf("WorldEdit %s brush", cfg.Type)
	return i.WithValue(keys.BrushConfigKey, string(data)).WithCustomName(name), nil
}

// ConfigFromItem reads brush JSON from an item value.
func ConfigFromItem(i item.Stack) (service.BrushConfig, bool) {
	v, ok := i.Value(keys.BrushConfigKey)
	if !ok {
		return service.BrushConfig{}, false
	}
	var raw string
	switch t := v.(type) {
	case string:
		raw = t
	case []byte:
		raw = string(t)
	default:
		return service.BrushConfig{}, false
	}
	var cfg service.BrushConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return service.BrushConfig{}, false
	}
	return cfg, true
}

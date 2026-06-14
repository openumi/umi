package log

import (
	"encoding/json"

	umi "github.com/openumi/umi"
)

func init() {
	umi.RegisterModule(new(LogModule))
}

type LogModule struct {
	Tag   string `json:"tag"`
	Level string `json:"level"`
}

func (l *LogModule) UniModule() umi.ModuleInfo {
	return umi.ModuleInfo{
		ID:  "log",
		New: func() umi.Module { return new(LogModule) },
	}
}

func (l *LogModule) Configure(ctx *umi.ConfigContext, raw json.RawMessage) error {
	return umi.ConfigureModule(ctx, l, raw)
}

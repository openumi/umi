package log

import (
	umi "github.com/openumi/umi"
)

func init() {
	umi.RegisterModule(new(LogModule))
}

type LogModule struct {
	Level string `json:"level"`
}

func (l *LogModule) UniModule() umi.ModuleInfo {
	return umi.ModuleInfo{
		ID:  "log",
		New: func() umi.Module { return new(LogModule) },
	}
}

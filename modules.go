package umi

import (
	"encoding/json"
	"sync"
)

type Module interface {
	UniModule() ModuleInfo
}

type ModuleID string
type ModuleTag string

type ModuleInfo struct {
	ID  ModuleID
	New func() Module
}

type Configurable interface {
	Configure(ctx *ConfigContext, raw json.RawMessage) error
}

type Provisioner interface {
	Provision(ctx *Context) error
}

type Validator interface {
	Validate() error
}

type Cleaner interface {
	Cleanup() error
}

func RegisterModule(instance Module) {
	info := instance.UniModule()

	if info.ID == "" {
		panic("umi: RegisterModule called with empty ModuleID")
	}

	if info.New == nil {
		panic("umi: RegisterModule called with nil New func for module '" + string(info.ID) + "'")
	}

	if val := info.New(); val == nil {
		panic("umi: RegisterModule: ModuleInfo.New returned nil for module '" + string(info.ID) + "'")
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := moduleRegistry[info.ID]; exists {
		panic("umi: RegisterModule: duplicate registration of module '" + string(info.ID) + "'")
	}

	moduleRegistry[info.ID] = info
}

var (
	moduleRegistry = make(map[ModuleID]ModuleInfo)
	registryMu     sync.RWMutex
)

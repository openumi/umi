package umi

import (
	"encoding/json"
	"sync"
)

type Module interface {
	UmiModule() ModuleInfo
}

type ModuleID string

type ModuleInfo struct {
	ID  ModuleID
	New func() Module
}

type ModuleMap map[string]json.RawMessage

func RegisterModule(mod Module) {
	inst := mod.UmiModule()

	if inst.ID == "" {
		panic("error: RegisterModule called with empty Module ID")
	}

	if inst.New == nil {
		panic("error: RegisterModule called with nil New func")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[inst.ID]; dup {
		panic("error: duplicate registration of id '" + inst.ID + "'")
	}

	registry[inst.ID] = inst
}

var (
	registry   = map[ModuleID]ModuleInfo{}
	registryMu sync.RWMutex
)

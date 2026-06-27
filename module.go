package umi

import (
	"sync"

	"github.com/openumi/fx"
)

// ============================================================================
// 2. MODULE REGISTRY (Caddy-style 编译裁剪扩展机制)
// ============================================================================

// ModuleFactory 接收原始的配置文本，返回一个未启动的 fx.App 实例
type ModuleFactory func(instanceID string, rawConfig []byte, bus *ExchangeBus) *fx.App

var (
	modulesMu sync.RWMutex
	modules   = make(map[string]ModuleFactory)
)

// RegisterModule 由各模块在 init() 中调用
func RegisterModule(name string, factory ModuleFactory) {
	modulesMu.Lock()
	defer modulesMu.Unlock()
	modules[name] = factory
}

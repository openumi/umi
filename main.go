package umi

import (
	"context"
	"fmt"
	"time"

	"github.com/openumi/fx"
)

// ============================================================================
// 3. CONFIG LAYER & BOOTSTRAP (运行时装配引擎)
// ============================================================================

type Runtime struct {
	bus      *ExchangeBus
	fxApps   []*fx.App
	stopFunc func()
}

func Bootstrap(configs []InstanceConfig) (*Runtime, error) {
	bus := NewExchangeBus()
	runtime := &Runtime{
		bus: bus,
	}

	modulesMu.RLock()
	defer modulesMu.RUnlock()

	for _, icfg := range configs {
		factory, exists := modules[icfg.Type]
		if !exists {
			return nil, fmt.Errorf("unknown module type: %s", icfg.Type)
		}

		// 核心纽带：通过闭包将通用的 runtime 基础设施工厂化传入各自隔离的 Fx 中
		app := factory(icfg.ID, []byte(icfg.Config), bus)
		runtime.fxApps = append(runtime.fxApps, app)
	}

	return runtime, nil
}

func (r *Runtime) Start() error {
	ctx := context.Background()
	for _, app := range r.fxApps {
		if err := app.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 倒序关闭所有 Fx App
	for i := len(r.fxApps) - 1; i >= 0; i-- {
		_ = r.fxApps[i].Stop(ctx)
	}
}

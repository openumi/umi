package umi

import "sync"

type Controller struct {
	mu      sync.Mutex
	runtime *Runtime
}

func (c *Controller) Reload(configData []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	newRuntime := NewRuntime()

	// Phase 1: 全量配置与实例登记
	if err := newRuntime.Init(configData); err != nil {
		return err
	}

	// Phase 2: 全量图状跨枝装配与验证
	if err := newRuntime.InitAll(); err != nil {
		return err
	}

	// 替换老运行实体
	if c.runtime != nil {
		_ = c.runtime.Stop()
	}
	c.runtime = newRuntime
	return nil
}

func (c *Controller) GetRuntime() *Runtime {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.runtime
}

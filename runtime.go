package umi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Runtime struct {
	Instances map[ModuleTag]Module
}

func NewRuntime() *Runtime {
	return &Runtime{
		Instances: make(map[ModuleTag]Module),
	}
}

// Init Task 1 - Phase 1: 纯粹的配置打平解析与落户登记区
func (r *Runtime) Init(configData []byte) error {
	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(configData, &topLevel); err != nil {
		return fmt.Errorf("failed to parse top-level flat config: %w", err)
	}

	ctx := &ConfigContext{runtime: r}

	// 无序线性遍历顶层平铺的所有 Key
	for key, raw := range topLevel {
		trimmed := strings.TrimSpace(string(raw))
		if len(trimmed) == 0 {
			continue
		}

		switch trimmed[0] {
		case '{':
			// 单体 App
			if err := r.parseAndRegisterTop(ctx, key, key, raw); err != nil {
				return err
			}
		case '[':
			// 多态 App 数组阵列开箱
			var list []json.RawMessage
			if err := json.Unmarshal(raw, &list); err != nil {
				return err
			}
			for idx, itemRaw := range list {
				defaultTag := fmt.Sprintf("%s.%02d", key, idx+1)
				if err := r.parseAndRegisterTop(ctx, key, defaultTag, itemRaw); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// instantiateAndRegister 共享核心：registry 查找 → New → Configure → 落户全局池
func (r *Runtime) instantiateAndRegister(ctx *ConfigContext, fullID ModuleID, defaultTag string, raw json.RawMessage, shadow map[string]json.RawMessage) (Module, error) {
	registryMu.RLock()
	info, exists := moduleRegistry[fullID]
	registryMu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unknown module type: '%s'", fullID)
	}

	instance := info.New()

	finalTag := string(defaultTag)
	if rawTag, ok := shadow["tag"]; ok {
		_ = json.Unmarshal(rawTag, &finalTag)
	}

	if cfg, ok := instance.(Configurable); ok {
		if err := cfg.Configure(ctx, raw); err != nil {
			return nil, err
		}
	}

	// 自动填充 Tag：如果实例的 Tag 字段为空，填入最终确定的 tag
	instVal := reflect.ValueOf(instance).Elem()
	if tf := instVal.FieldByName("Tag"); tf.IsValid() && tf.Kind() == reflect.String && tf.String() == "" {
		tf.SetString(finalTag)
	}

	if _, exists := r.Instances[ModuleTag(finalTag)]; exists {
		fmt.Printf("[Warning] flat promotion: overwriting existing tag '%s' with module '%s'\n", finalTag, fullID)
	}
	r.Instances[ModuleTag(finalTag)] = instance
	return instance, nil
}

func (r *Runtime) parseAndRegisterTop(ctx *ConfigContext, moduleType string, defaultTag string, raw json.RawMessage) error {
	var shadow map[string]json.RawMessage
	if err := json.Unmarshal(raw, &shadow); err != nil {
		return err
	}
	_, err := r.instantiateAndRegister(ctx, ModuleID(moduleType), defaultTag, raw, shadow)
	return err
}

// InitAll Task 1 - Phase 2 & Task 4: 跨分支组装期，精准且仅触发一次
func (r *Runtime) InitAll() error {
	ctx := &Context{runtime: r}

	// 1. 一阶全网线性遍历：触发 Provision 组装依赖
	var provisioned []ModuleTag
	for tag, inst := range r.Instances {
		if prov, ok := inst.(Provisioner); ok {
			if err := prov.Provision(ctx); err != nil {
				// 回滚：对已 Provision 成功的模块调用 Cleanup
				for _, prevTag := range provisioned {
					if prevInst, ok := r.Instances[prevTag]; ok {
						if cleaner, ok := prevInst.(Cleaner); ok {
							_ = cleaner.Cleanup()
						}
					}
				}
				return fmt.Errorf("provision failed for tag '%s': %w", tag, err)
			}
			provisioned = append(provisioned, tag)
		}
	}

	// 2. 二阶全网线性遍历：触发统一校验
	for _, inst := range r.Instances {
		if val, ok := inst.(Validator); ok {
			if err := val.Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Stop Task 4: 统一幂等销毁
func (r *Runtime) Stop() error {
	var errs []error
	for tag, inst := range r.Instances {
		if cleaner, ok := inst.(Cleaner); ok {
			if err := cleaner.Cleanup(); err != nil {
				errs = append(errs, fmt.Errorf("cleanup failed for tag '%s': %w", tag, err))
			}
		}
	}
	return errors.Join(errs...)
}

// FindInstance 供 Phase 1 / Phase 2 内部基础查找使用
func (r *Runtime) FindInstance(tag ModuleTag) (Module, error) {
	mod, exists := r.Instances[tag]
	if !exists {
		return nil, fmt.Errorf("instance '%s' not found in global runtime pool", tag)
	}
	return mod, nil
}

package umi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

// Init Phase 1: flat config parsing and module registration
func (r *Runtime) Init(configData []byte) error {
	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(configData, &topLevel); err != nil {
		return fmt.Errorf("failed to parse top-level flat config: %w", err)
	}

	ctx := &ConfigContext{runtime: r}

	// iterate all top-level keys in the flat config
	for key, raw := range topLevel {
		trimmed := strings.TrimSpace(string(raw))
		if len(trimmed) == 0 {
			continue
		}

		switch trimmed[0] {
		case '{':
			// single module object
			if err := r.parseAndRegisterTop(ctx, key, key, raw); err != nil {
				return err
			}
		case '[':
			// module array — multiple instances of the same type
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

// instantiateAndRegister shared core: registry lookup → New → Configure → register into pool
func (r *Runtime) instantiateAndRegister(ctx *ConfigContext, fullID ModuleID, defaultTag string, raw json.RawMessage, shadow map[string]json.RawMessage) (Module, error) {
	registryMu.RLock()
	info, exists := moduleRegistry[fullID]
	registryMu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("unknown module type: '%s'", fullID)
	}

	instance := info.New()

	// framework-level JSON injection: ensure all Modules get their data from config
	if len(raw) > 0 {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(instance); err != nil {
			return nil, fmt.Errorf("decoding module '%s': %w", fullID, err)
		}
	}

	finalTag := string(defaultTag)
	if rawTag, ok := shadow["tag"]; ok {
		_ = json.Unmarshal(rawTag, &finalTag)
	}

	// inject module identity into context before Configure
	ctx.selfID = fullID
	ctx.selfTag = ModuleTag(finalTag)

	if cfg, ok := instance.(Configurable); ok {
		if err := cfg.Configure(ctx, raw); err != nil {
			return nil, err
		}
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

// InitAll Phase 2: cross-branch assembly — Provision and Validate
func (r *Runtime) InitAll() error {
	ctx := &Context{runtime: r}

	// round 1: Provision — build inter-module dependencies
	var provisioned []ModuleTag
	for tag, inst := range r.Instances {
		if prov, ok := inst.(Provisioner); ok {
			// inject module identity before Provision
			ctx.selfID = inst.UniModule().ID
			ctx.selfTag = tag

			if err := prov.Provision(ctx); err != nil {
				// rollback: cleanup already provisioned modules
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

	// round 2: Validate
	for _, inst := range r.Instances {
		if val, ok := inst.(Validator); ok {
			if err := val.Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Stop idempotently cleans up all modules
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

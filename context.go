package umi

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type Context struct {
	context.Context

	moduleInstances map[string][]Module
	cfg             *Config
	cleanupFuncs    []func()
	cancelFunc      context.CancelCauseFunc
}

func NewContext(ctx Context) (Context, context.CancelFunc) {
	newCtx, cancelCause := NewContextWithCause(ctx)
	return newCtx, func() { cancelCause(nil) }
}

func NewContextWithCause(ctx Context) (Context, context.CancelCauseFunc) {
	newCtx := Context{
		moduleInstances: make(map[string][]Module),
		cfg:             ctx.cfg,
	}

	c, cancel := context.WithCancelCause(ctx.Context)

	wrappedCancel := func(cause error) {
		cancel(cause)
		for _, f := range ctx.cleanupFuncs {
			f()
		}
		for modName, instances := range newCtx.moduleInstances {
			for _, inst := range instances {
				if cu, ok := inst.(CleanerUpper); ok {
					if err := cu.Cleanup(); err != nil {
						fmt.Printf("[ERROR] %s (%p): cleanup: %v\n", modName, inst, err)
					}
				}
			}
		}
	}

	newCtx.Context = c
	newCtx.cancelFunc = wrappedCancel
	return newCtx, wrappedCancel
}

func (ctx *Context) OnCancel(f func()) {
	ctx.cleanupFuncs = append(ctx.cleanupFuncs, f)
}

func (ctx Context) App(name string) (any, error) {
	if app, ok := ctx.cfg.apps[name]; ok {
		return app, nil
	}

	appRaw := ctx.cfg.AppsRaw[name]
	modVal, err := ctx.LoadModuleByID(name, appRaw)
	if err != nil {
		return nil, fmt.Errorf("loading %s app module: %v", name, err)
	}
	if appRaw != nil {
		ctx.cfg.AppsRaw[name] = nil // allow GC
	}

	return modVal, nil
}

func (ctx Context) LoadModuleByID(id string, rawMsg json.RawMessage) (any, error) {
	modID := ModuleID(id)
	registryMu.RLock()
	modInfo, ok := registry[modID]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown module: %s", id)
	}

	val := modInfo.New()

	if rv := reflect.ValueOf(val); rv.Kind() != reflect.Pointer {
		val = reflect.New(rv.Type()).Elem().Addr().Interface().(Module)
	}

	if len(rawMsg) > 0 {
		if err := json.Unmarshal(rawMsg, &val); err != nil {
			return nil, fmt.Errorf("decoding module config: %s: %v", id, err)
		}
	}

	if val == nil {
		return nil, fmt.Errorf("module value cannot be null")
	}

	if appModule, ok := val.(App); ok {
		ctx.cfg.apps[id] = appModule
	}

	if prov, ok := val.(Provisioner); ok {
		err := prov.Provision(ctx)
		if err != nil {
			if cleaner, ok := val.(CleanerUpper); ok {
				_ = cleaner.Cleanup()
			}
			return nil, fmt.Errorf("provisioning module '%s': %v", id, err)
		}
	}

	if validator, ok := val.(Validator); ok {
		err := validator.Validate()
		if err != nil {
			if cleaner, ok := val.(CleanerUpper); ok {
				_ = cleaner.Cleanup()
			}
			return nil, fmt.Errorf("validating module '%s': %v", id, err)
		}
	}

	// 存入 moduleInstances
	ctx.moduleInstances[id] = append(ctx.moduleInstances[id], val)

	return val, nil
}

func (ctx Context) loadModuleInline(inlinekey, namespace string, rawMsg json.RawMessage) (any, error) {
	var partial map[string]json.RawMessage
	if err := json.Unmarshal(rawMsg, &partial); err != nil {
		return nil, err
	}

	moduleName := string(partial[inlinekey])
	moduleName = moduleName[1 : len(moduleName)-1]

	var id string
	if namespace != "" {
		id = namespace + "." + moduleName
	} else {
		id = moduleName
	}

	delete(partial, inlinekey)
	cleanedRaw, err := json.Marshal(partial)
	if err != nil {
		return nil, err
	}

	return ctx.LoadModuleByID(id, cleanedRaw)
}

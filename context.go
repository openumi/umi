package umi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ConfigContext 配置期上下文——Phase 1 中传递给 Configure 方法。
// 持有当前正在配置的模块的身份信息。
type ConfigContext struct {
	runtime *Runtime
	selfID  ModuleID
	selfTag ModuleTag
}

// ID 返回当前正在配置的模块的 ModuleID。
func (c *ConfigContext) ID() ModuleID { return c.selfID }

// Tag 返回当前正在配置的模块的运行时 Tag。
func (c *ConfigContext) Tag() ModuleTag { return c.selfTag }

// Context 装配期上下文——Phase 2 中传递给 Provision 方法。
// 持有当前正在装配的模块的身份信息。
type Context struct {
	runtime *Runtime
	selfID  ModuleID
	selfTag ModuleTag
}

// ID 返回当前正在装配的模块的 ModuleID。
func (c *Context) ID() ModuleID { return c.selfID }

// Tag 返回当前正在装配的模块的运行时 Tag。
func (c *Context) Tag() ModuleTag { return c.selfTag }

// FindInstance 按 Tag 查找运行时实例，支持命名空间安全门禁。
// 如果 expectedNamespace 非空，会校验目标实例的 ModuleID 是否以该前缀开头。
func (c *Context) FindInstance(expectedNamespace string, tag string) (Module, error) {
	mod, exists := c.runtime.Instances[ModuleTag(tag)]
	if !exists {
		return nil, fmt.Errorf("instance not found with tag: %s", tag)
	}

	actualID := string(mod.UniModule().ID)
	if expectedNamespace != "" && !strings.HasPrefix(actualID, expectedNamespace) {
		return nil, fmt.Errorf("security error: tag '%s' is a '%s', but expected namespace prefix '%s'", tag, actualID, expectedNamespace)
	}

	return mod, nil
}

// ConfigureModule 反射驱动的子模块自动加载引擎。
//
// 扫描 mod 中带有 umi 结构体标签的字段（必须是 json.RawMessage 或 []json.RawMessage），
// 对每个字段从 registry 查找模块类型、实例化、配置，并写入对应的目标字段。
//
// 注意：框架在调用 Configure 之前已统一完成 json.Unmarshal，
// 此函数不再重复反序列化纯数据字段。
func ConfigureModule(ctx *ConfigContext, mod any, raw json.RawMessage) error {
	val := reflect.ValueOf(mod).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		umiTag := field.Tag.Get("umi")
		if umiTag == "" {
			continue
		}

		tagMap := parseUmiTag(umiTag)
		namespace := tagMap["namespace"]
		if namespace == "" {
			return fmt.Errorf("umi: struct tag on field '%s.%s' requires 'namespace=...'", typ.Name(), field.Name)
		}
		inlineKey := tagMap["inline_key"]

		jsonTag := field.Tag.Get("json")
		jsonFieldName := strings.Split(jsonTag, ",")[0]
		if jsonFieldName == "" {
			jsonFieldName = field.Name
		}

		fieldVal := val.Field(i)

		// 影子字段：去掉 "Raw" 后缀后写入解码后的模块实例
		targetFieldName := strings.TrimSuffix(field.Name, "Raw")
		targetField := val.FieldByName(targetFieldName)

		// 单体子模块：json.RawMessage
		if field.Type == reflect.TypeOf(json.RawMessage{}) {
			rawMsg := fieldVal.Interface().(json.RawMessage)
			if len(rawMsg) == 0 {
				continue
			}

			liveMod, err := ctx.loadModuleInline(rawMsg, namespace, inlineKey, jsonFieldName)
			if err != nil {
				return err
			}

			if liveMod != nil && targetField.IsValid() && targetField.CanSet() && targetField.UnsafeAddr() != fieldVal.UnsafeAddr() {
				targetField.Set(reflect.ValueOf(liveMod))
			}
		}

		// 子模块数组：[]json.RawMessage
		if field.Type == reflect.TypeOf([]json.RawMessage{}) {
			rawSlice := fieldVal.Interface().([]json.RawMessage)
			for idx, rawMsg := range rawSlice {
				if len(rawMsg) == 0 {
					continue
				}

				defaultItemTag := fmt.Sprintf("%s.%d", jsonFieldName, idx)
				liveMod, err := ctx.loadModuleInline(rawMsg, namespace, inlineKey, defaultItemTag)
				if err != nil {
					return err
				}

				if liveMod != nil && targetField.IsValid() && targetField.CanSet() && targetField.UnsafeAddr() != fieldVal.UnsafeAddr() {
					targetField.Set(reflect.Append(targetField, reflect.ValueOf(liveMod)))
				}
			}
		}
	}
	return nil
}

// loadModuleInline 将一段 JSON 解析为模块实例。
//
// 分流逻辑：
//   - 字符串（"..."）：方案 A，延迟绑定，留到 Phase 2 解析引用，返回 nil
//   - 对象（{...}）：方案 B，根据 inline_key 或默认字段名确定模块名，实例化并注册
func (ctx *ConfigContext) loadModuleInline(raw json.RawMessage, namespace, inlineKey, defaultTag string) (Module, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}

	// 方案 A：双引号开头表示纯字符串引用别名，Phase 1 不解析
	if trimmed[0] == '"' {
		return nil, nil
	}

	// 方案 B：JSON 对象，需要实例化子模块
	var shadow map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &shadow); err != nil {
		return nil, err
	}

	var moduleName string
	if inlineKey != "" {
		if rawType, ok := shadow[inlineKey]; ok {
			_ = json.Unmarshal(rawType, &moduleName)
		}
	}
	if moduleName == "" {
		moduleName = defaultTag
	}

	fullID := ModuleID(moduleName)
	if namespace != "" {
		fullID = ModuleID(namespace + "." + moduleName)
	}

	return ctx.runtime.instantiateAndRegister(ctx, fullID, defaultTag, trimmed, shadow)
}

func parseUmiTag(tag string) map[string]string {
	res := make(map[string]string)
	parts := strings.FieldsSeq(tag)
	for part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			res[kv[0]] = kv[1]
		}
	}
	return res
}

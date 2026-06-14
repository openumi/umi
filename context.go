package umi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ConfigContext 阶段一：配置期上下文
type ConfigContext struct {
	runtime *Runtime
}

// Context 阶段二：装配期上下文
type Context struct {
	runtime *Runtime
}

// FindInstance Task 3: 落实跨分支命名空间安全门禁
func (c *Context) FindInstance(expectedNamespace string, tag string) (Module, error) {
	mod, exists := c.runtime.Instances[ModuleTag(tag)]
	if !exists {
		return nil, fmt.Errorf("instance not found with tag: %s", tag)
	}

	// 强类型安全门禁高亮执法
	actualID := string(mod.UniModule().ID)
	if expectedNamespace != "" && !strings.HasPrefix(actualID, expectedNamespace) {
		return nil, fmt.Errorf("security error: tag '%s' is a '%s', but expected namespace prefix '%s'", tag, actualID, expectedNamespace)
	}

	return mod, nil
}

// ConfigureModule Task 0 & Task 2: 自动化反射清洗引擎，支持打平提升
func ConfigureModule(ctx *ConfigContext, mod any, raw json.RawMessage) error {
	// 首先让原生强类型纯数据字段就位
	if err := json.Unmarshal(raw, mod); err != nil {
		return err
	}

	val := reflect.ValueOf(mod).Elem()
	typ := val.Type()

	// 强制校验：每个 Module 必须声明 Tag string 字段
	tagField := val.FieldByName("Tag")
	if !tagField.IsValid() || tagField.Kind() != reflect.String {
		return fmt.Errorf("umi: module type '%s' must have a 'Tag string' json field", typ.Name())
	}

	// 扫描带有 umi 标签的 blind box 字段
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

		// 承载反序列化出的真实结构体指针或切片（支持可选的影子字段映射机制）
		targetFieldName := strings.TrimSuffix(field.Name, "Raw")
		targetField := val.FieldByName(targetFieldName)

		// 1. 处理单体盲盒 (json.RawMessage)
		if field.Type == reflect.TypeOf(json.RawMessage{}) {
			rawMsg := fieldVal.Interface().(json.RawMessage)
			if len(rawMsg) == 0 {
				continue
			}

			liveMod, err := ctx.loadModuleInline(rawMsg, namespace, inlineKey, jsonFieldName)
			if err != nil {
				return err
			}

			// 注意：如果是方案 A 返回了 nil，这里不覆盖 targetField
			// 同时避免向自身（json.RawMessage 字段）写入 Module 实例
			if liveMod != nil && targetField.IsValid() && targetField.CanSet() && targetField.UnsafeAddr() != fieldVal.UnsafeAddr() {
				targetField.Set(reflect.ValueOf(liveMod))
			}
		}

		// 2. 处理切片盲盒 ([]json.RawMessage)
		if field.Type == reflect.TypeOf([]json.RawMessage{}) {
			rawSlice := fieldVal.Interface().([]json.RawMessage)
			for idx, rawMsg := range rawSlice {
				if len(rawMsg) == 0 {
					continue
				}

				// 数组项隐式推导默认 tag 别名
				defaultItemTag := fmt.Sprintf("%s.%d", jsonFieldName, idx)
				liveMod, err := ctx.loadModuleInline(rawMsg, namespace, inlineKey, defaultItemTag)
				if err != nil {
					return err
				}

				// 同样避免向自身（[]json.RawMessage 字段）写入 Module 实例
				if liveMod != nil && targetField.IsValid() && targetField.CanSet() && targetField.UnsafeAddr() != fieldVal.UnsafeAddr() {
					targetField.Set(reflect.Append(targetField, reflect.ValueOf(liveMod)))
				}
			}
		}
	}
	return nil
}

// loadModuleInline Task 2: 延迟满足核心分流器
func (ctx *ConfigContext) loadModuleInline(raw json.RawMessage, namespace, inlineKey, defaultTag string) (Module, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}

	// 🚀 方案 A 拦截：首字节为双引号 `"` 代表是纯字符串引用别名
	if trimmed[0] == '"' {
		// 延迟满足：在 Phase 1 (Configure) 绝对不进行 FindInstance 检索，直接放行返回 nil
		return nil, nil
	}

	// 📦 方案 B 拦截：首字节为 `{` 走全量对象构建
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

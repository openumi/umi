package umi

// InstanceConfig 模拟人类可读的配置文件结构
type InstanceConfig struct {
	ID     string `yaml:"id"`
	Type   string `yaml:"type"`
	Config string `yaml:"config"` // 实际项目中为 json.RawMessage 或 yaml.Node
}

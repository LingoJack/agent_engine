package conf

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProviderConfig 定义单个 LLM 提供商的配置
type ProviderConfig struct {
	Name    string   `yaml:"name"`    // 提供商名称
	ApiKey  string   `yaml:"api_key"` // API密钥
	BaseUrl string   `yaml:"base_url"` // 基础URL
	Model   []string `yaml:"model"`   // 支持的模型列表
}

// Config 定义整体配置结构
type Config struct {
	Provider []ProviderConfig `yaml:"provider"` // 提供商列表
}

// LoadConfig 从指定路径加载 YAML 配置文件
// 参数:
//   - configPath: 配置文件路径
// 返回:
//   - *Config: 配置对象指针
//   - error: 错误信息
func LoadConfig(configPath string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 YAML
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

// GetDefaultProvider 获取默认的提供商配置（第一个）
// 返回:
//   - *ProviderConfig: 提供商配置指针
//   - error: 错误信息
func (c *Config) GetDefaultProvider() (*ProviderConfig, error) {
	if len(c.Provider) == 0 {
		return nil, fmt.Errorf("配置文件中没有提供商配置")
	}
	return &c.Provider[0], nil
}

// GetProviderByName 根据名称获取提供商配置
// 参数:
//   - name: 提供商名称
// 返回:
//   - *ProviderConfig: 提供商配置指针
//   - error: 错误信息
func (c *Config) GetProviderByName(name string) (*ProviderConfig, error) {
	for i := range c.Provider {
		if c.Provider[i].Name == name {
			return &c.Provider[i], nil
		}
	}
	return nil, fmt.Errorf("未找到名为 %s 的提供商配置", name)
}

// GetDefaultModel 获取提供商的默认模型（第一个）
// 返回:
//   - string: 模型名称
//   - error: 错误信息
func (p *ProviderConfig) GetDefaultModel() (string, error) {
	if len(p.Model) == 0 {
		return "", fmt.Errorf("提供商 %s 没有配置模型", p.Name)
	}
	return p.Model[0], nil
}

// HasModel 检查提供商是否支持指定的模型
// 参数:
//   - modelId: 模型ID
// 返回:
//   - bool: 是否支持该模型
func (p *ProviderConfig) HasModel(modelId string) bool {
	for _, m := range p.Model {
		if m == modelId {
			return true
		}
	}
	return false
}
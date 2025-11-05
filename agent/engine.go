package agent

import (
	"agent_engine/conf"
	"context"
	"fmt"
	"path/filepath"
)

var (
	// 处理器映射，根据事件类型查找对应的处理接口实现
	eventHandlerMap = map[string]EventHandler{
		"query": &QueryHandler{},
		"list":  &ListHandler{}, // 列出所有提供商和模型
	}
)

// EventHandler 定义事件处理接口
type EventHandler interface {
	Handle(ctx context.Context, engine *Engine, params string, event string) (rsp any, err error)
}

// Engine 代理引擎，后续可能接入 MCP
// 注意：ApiKey 为私有字段以保护敏感信息，通过 GetApiKey() 方法访问
type Engine struct {
	// 公开字段
	ModelId string `json:"model_id"` // 当前使用的模型ID
	BaseUrl string `json:"base_url"` // 当前使用的基础URL

	// 私有字段
	apiKey       string       // 当前使用的API密钥（敏感信息）
	configPath   string       // 配置文件路径
	config       *conf.Config // 配置对象
	providerName string       // 当前提供商名称
}

// GetApiKey 获取 API 密钥（提供受控访问）
func (engine *Engine) GetApiKey() string {
	return engine.apiKey
}

// NewEngineFromConfig 从配置文件创建 Engine 实例
// 参数:
//   - configPath: 配置文件路径（支持相对路径和绝对路径）
//   - providerName: 提供商名称，如果为空则使用默认提供商（第一个）
//   - modelId: 模型ID，如果为空则使用提供商的默认模型（第一个）
// 返回:
//   - *Engine: Engine 实例指针
//   - error: 错误信息
func NewEngineFromConfig(configPath string, providerName string, modelId string) (*Engine, error) {
	// 将配置文件路径转换为绝对路径
	// 如果传入的是相对路径，会基于当前工作目录转换为绝对路径
	// 如果传入的已经是绝对路径，则保持不变
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("转换配置文件路径为绝对路径失败: %w", err)
	}

	// 加载配置文件（使用原始路径加载，因为相对路径也能正常工作）
	config, err := conf.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}

	// 获取提供商配置
	var provider *conf.ProviderConfig
	if providerName == "" {
		// 使用默认提供商（第一个）
		provider, err = config.GetDefaultProvider()
		if err != nil {
			return nil, fmt.Errorf("获取默认提供商失败: %w", err)
		}
	} else {
		// 使用指定的提供商
		provider, err = config.GetProviderByName(providerName)
		if err != nil {
			return nil, fmt.Errorf("获取提供商 %s 失败: %w", providerName, err)
		}
	}

	// 获取模型ID
	var finalModelId string
	if modelId == "" {
		// 使用默认模型（第一个）
		finalModelId, err = provider.GetDefaultModel()
		if err != nil {
			return nil, fmt.Errorf("获取默认模型失败: %w", err)
		}
	} else {
		// 验证指定的模型是否存在
		if !provider.HasModel(modelId) {
			return nil, fmt.Errorf("提供商 %s 不支持模型 %s", provider.Name, modelId)
		}
		finalModelId = modelId
	}

	// 创建 Engine 实例
	engine := &Engine{
		ModelId:      finalModelId,
		BaseUrl:      provider.BaseUrl,
		apiKey:       provider.ApiKey,
		configPath:   absConfigPath, // 存储绝对路径
		config:       config,
		providerName: provider.Name,
	}

	return engine, nil
}

// GetAvailableProviders 获取所有可用的提供商列表
// 返回:
//   - []string: 提供商名称列表
//   - error: 错误信息
func (engine *Engine) GetAvailableProviders() ([]string, error) {
	if engine.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	providers := make([]string, 0, len(engine.config.Provider))
	for _, p := range engine.config.Provider {
		providers = append(providers, p.Name)
	}

	return providers, nil
}

// GetAvailableModels 获取当前提供商的所有可用模型列表
// 返回:
//   - []string: 模型ID列表
//   - error: 错误信息
func (engine *Engine) GetAvailableModels() ([]string, error) {
	if engine.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	// 获取当前提供商配置
	provider, err := engine.config.GetProviderByName(engine.providerName)
	if err != nil {
		return nil, fmt.Errorf("获取当前提供商配置失败: %w", err)
	}

	return provider.Model, nil
}

// GetAllModels 获取所有提供商的所有模型列表（带提供商信息）
// 返回:
//   - map[string][]string: 提供商名称到模型列表的映射
//   - error: 错误信息
func (engine *Engine) GetAllModels() (map[string][]string, error) {
	if engine.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	allModels := make(map[string][]string)
	for _, p := range engine.config.Provider {
		allModels[p.Name] = p.Model
	}

	return allModels, nil
}

// SwitchProvider 切换到指定的提供商
// 参数:
//   - providerName: 提供商名称
//   - modelId: 模型ID，如果为空则使用该提供商的默认模型（第一个）
// 返回:
//   - error: 错误信息
func (engine *Engine) SwitchProvider(providerName string, modelId string) error {
	if engine.config == nil {
		return fmt.Errorf("配置未加载")
	}

	// 获取指定的提供商配置
	provider, err := engine.config.GetProviderByName(providerName)
	if err != nil {
		return fmt.Errorf("获取提供商 %s 失败: %w", providerName, err)
	}

	// 获取模型ID
	var finalModelId string
	if modelId == "" {
		// 使用默认模型（第一个）
		finalModelId, err = provider.GetDefaultModel()
		if err != nil {
			return fmt.Errorf("获取默认模型失败: %w", err)
		}
	} else {
		// 验证指定的模型是否存在
		if !provider.HasModel(modelId) {
			return fmt.Errorf("提供商 %s 不支持模型 %s", provider.Name, modelId)
		}
		finalModelId = modelId
	}

	// 更新 Engine 配置
	engine.providerName = provider.Name
	engine.apiKey = provider.ApiKey
	engine.BaseUrl = provider.BaseUrl
	engine.ModelId = finalModelId

	return nil
}

// SwitchModel 在当前提供商下切换模型
// 参数:
//   - modelId: 模型ID
// 返回:
//   - error: 错误信息
func (engine *Engine) SwitchModel(modelId string) error {
	if engine.config == nil {
		return fmt.Errorf("配置未加载")
	}

	// 获取当前提供商配置
	provider, err := engine.config.GetProviderByName(engine.providerName)
	if err != nil {
		return fmt.Errorf("获取当前提供商配置失败: %w", err)
	}

	// 验证模型是否存在
	if !provider.HasModel(modelId) {
		return fmt.Errorf("当前提供商 %s 不支持模型 %s", engine.providerName, modelId)
	}

	// 更新模型ID
	engine.ModelId = modelId

	return nil
}

// GetCurrentProviderName 获取当前提供商名称
// 返回:
//   - string: 提供商名称
func (engine *Engine) GetCurrentProviderName() string {
	return engine.providerName
}

// GetConfigPath 获取配置文件路径
// 返回:
//   - string: 配置文件绝对路径
func (engine *Engine) GetConfigPath() string {
	return engine.configPath
}

// GetAllProvidersInfo 获取所有提供商的详细信息（包括 base_url）
// 返回:
//   - map[string]*conf.ProviderConfig: 提供商名称到配置的映射
//   - error: 错误信息
func (engine *Engine) GetAllProvidersInfo() (map[string]*conf.ProviderConfig, error) {
	if engine.config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	providersInfo := make(map[string]*conf.ProviderConfig)
	for i := range engine.config.Provider {
		p := &engine.config.Provider[i]
		providersInfo[p.Name] = p
	}

	return providersInfo, nil
}

// DispatchAndHandle 分发和处理
// 参数:
//   - ctx: 上下文
//   - params: 参数字符串
//   - event: 事件类型
// 返回:
//   - rsp: 响应数据
//   - match: 是否匹配到处理器
//   - err: 错误信息
func (engine *Engine) DispatchAndHandle(ctx context.Context, params string, event string) (rsp any, match bool, err error) {
	match = false
	if handler, ok := eventHandlerMap[event]; ok {
		match = true
		rsp, err = handler.Handle(ctx, engine, params, event)
		return
	}
	return
}
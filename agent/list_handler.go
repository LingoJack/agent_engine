package agent

import (
	"context"
)

// ListHandler 实现 EventHandler 接口，处理列表查询事件
// 用于列出所有可用的提供商和模型信息
type ListHandler struct{}

// Handle 处理 list 命令，返回所有提供商和模型的信息
// 参数:
//   - ctx: 上下文
//   - engine: Engine 实例
//   - params: 参数（list 命令不需要参数，可以忽略）
//   - event: 事件类型
// 返回:
//   - rsp: 包含所有提供商和模型信息的响应
//   - err: 错误信息
func (h *ListHandler) Handle(ctx context.Context, engine *Engine, params string, event string) (rsp any, err error) {
	// 获取所有提供商列表
	providers, err := engine.GetAvailableProviders()
	if err != nil {
		return nil, err
	}

	// 获取所有提供商的模型信息
	allModels, err := engine.GetAllModels()
	if err != nil {
		return nil, err
	}

	// 获取所有提供商的详细信息（包括 base_url）
	allProvidersInfo, err := engine.GetAllProvidersInfo()
	if err != nil {
		return nil, err
	}

	// 获取当前使用的提供商和模型
	currentProvider := engine.GetCurrentProviderName()
	currentModel := engine.ModelId
	currentBaseUrl := engine.BaseUrl

	// 获取配置文件路径
	configPath := engine.GetConfigPath()

	// 构建详细的提供商信息列表
	type ProviderInfo struct {
		Name      string   `json:"name"`       // 提供商名称
		BaseUrl   string   `json:"base_url"`   // 提供商的 base_url
		Models    []string `json:"models"`     // 该提供商支持的模型列表
		IsCurrent bool     `json:"is_current"` // 是否为当前使用的提供商
	}

	providerInfos := make([]ProviderInfo, 0, len(providers))
	for _, providerName := range providers {
		// 从详细信息中获取 base_url
		baseUrl := ""
		if providerConfig, ok := allProvidersInfo[providerName]; ok {
			baseUrl = providerConfig.BaseUrl
		}

		providerInfos = append(providerInfos, ProviderInfo{
			Name:      providerName,
			BaseUrl:   baseUrl,
			Models:    allModels[providerName],
			IsCurrent: providerName == currentProvider,
		})
	}

	// 构建响应数据
	rsp = map[string]interface{}{
		"config_path":       configPath,         // 配置文件绝对路径
		"current_provider":  currentProvider,    // 当前提供商名称
		"current_model":     currentModel,       // 当前模型ID
		"current_base_url":  currentBaseUrl,     // 当前使用的 base_url
		"providers":         providerInfos,      // 所有提供商的详细信息
		"total_providers":   len(providers),     // 提供商总数
	}

	return rsp, nil
}
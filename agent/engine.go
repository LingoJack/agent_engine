package agent

import (
	"context"
)

var (
	// 处理器映射，根据事件类型查找对应的处理接口实现
	eventHandlerMap = map[string]EventHandler{
		"query": &QueryHandler{},
	}
)

// EventHandler 定义事件处理接口
type EventHandler interface {
	Handle(ctx context.Context, engine *Engine, params string, event string) (rsp any, err error)
}

// Engine 代理引擎，后续可能接入mcp
type Engine struct {
	ModelId string `json:"model_id"`
	ApiKey  string `json:"api_key"`
	BaseUrl string `json:"base_url"`
}

// NewEngineFromConfig 从配置文件创建 Engine 实例
func NewEngineFromConfig(configPath string, providerName string, modelId string) (*Engine, error) {
// ... existing code ...
}

// GetAvailableProviders 获取所有可用的提供商列表
func (engine *Engine) GetAvailableProviders() ([]string, error) {
// ... existing code ...
}

// GetAvailableModels 获取当前提供商的所有可用模型列表
func (engine *Engine) GetAvailableModels() ([]string, error) {
// ... existing code ...
}

// GetAllModels 获取所有提供商的所有模型列表（带提供商信息）
func (engine *Engine) GetAllModels() (map[string][]string, error) {
// ... existing code ...
}

// SwitchProvider 切换到指定的提供商
func (engine *Engine) SwitchProvider(providerName string, modelId string) error {
// ... existing code ...
}

// SwitchModel 在当前提供商下切换模型
func (engine *Engine) SwitchModel(modelId string) error {
// ... existing code ...
}

// GetCurrentProviderName 获取当前提供商名称
func (engine *Engine) GetCurrentProviderName() string {
// ... existing code ...
}

// DispatchAndHandle 分发和处理
func (engine *Engine) DispatchAndHandle(ctx context.Context, params string, event string) (rsp any, match bool, err error) {
	match = false
	if handler, ok := eventHandlerMap[event]; ok {
		match = true
		rsp, err = handler.Handle(ctx, engine, params, event)
		return
	}
	return
}
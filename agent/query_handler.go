package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// QueryHandler 实现 EventHandler 接口，处理查询事件
type QueryHandler struct{}

func (h *QueryHandler) Handle(ctx context.Context, engine *Engine, params string, event string) (rsp any, err error) {
	query := ""
	if json.Valid([]byte(params)) {
		type QueryReq struct {
			Query string `json:"query"`
		}
		var req QueryReq
		err = json.Unmarshal([]byte(params), &req)
		if err != nil {
			return
		}
		query = req.Query
	} else {
		query = params
	}

	// 保存原始模型ID，用于失败后恢复
	originalModelId := engine.ModelId
	defer func() {
		// 无论成功或失败，都恢复原始模型ID
		engine.ModelId = originalModelId
	}()

	// 获取当前提供商的所有可用模型
	availableModels, err := engine.GetAvailableModels()
	if err != nil {
		return nil, fmt.Errorf("获取可用模型列表失败: %w", err)
	}

	// 初始化随机数生成器
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 记录已尝试过的模型
	triedModels := make(map[string]bool)
	triedModels[originalModelId] = true

	// 最多尝试3个模型（包括当前模型）
	maxAttempts := 3
	if len(availableModels) < maxAttempts {
		maxAttempts = len(availableModels)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 第一次尝试使用原始模型，后续尝试随机选择未使用过的模型
		if attempt > 1 {
			// 获取未尝试过的模型列表
			untriedModels := make([]string, 0)
			for _, model := range availableModels {
				if !triedModels[model] {
					untriedModels = append(untriedModels, model)
				}
			}

			// 如果没有未尝试的模型了，退出循环
			if len(untriedModels) == 0 {
				log.Printf("[QueryHandler] 已尝试所有可用模型，无更多模型可轮换")
				break
			}

			// 随机选择一个未尝试过的模型
			newModelId := untriedModels[rnd.Intn(len(untriedModels))]
			log.Printf("[QueryHandler] 第 %d 次尝试：切换到模型 %s（提供商：%s）", attempt, newModelId, engine.GetCurrentProviderName())

			// 切换模型
			if err := engine.SwitchModel(newModelId); err != nil {
				log.Printf("[QueryHandler] 切换模型失败: %v", err)
				continue
			}

			// 标记该模型已尝试
			triedModels[newModelId] = true
		} else {
			log.Printf("[QueryHandler] 第 %d 次尝试：使用当前模型 %s（提供商：%s）", attempt, engine.ModelId, engine.GetCurrentProviderName())
		}

		// 尝试调用模型
		client := openai.NewClient(option.WithAPIKey(engine.GetApiKey()), option.WithBaseURL(engine.BaseUrl))
		completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage(query)},
			Model:    engine.ModelId,
		})

		if err != nil {
			lastErr = err
			log.Printf("[QueryHandler] 模型 %s 调用失败: %v", engine.ModelId, err)

			// 如果还有重试机会，继续下一次尝试
			if attempt < maxAttempts {
				continue
			}

			// 所有尝试都失败了，返回最后一次的错误
			return nil, fmt.Errorf("所有模型调用均失败，最后错误: %w", lastErr)
		}

		// 调用成功，记录日志并返回结果
		log.Printf("[QueryHandler] 模型 %s 调用成功（第 %d 次尝试）", engine.ModelId, attempt)
		log.Printf("[QueryHandler] raw json: %s", completion.RawJSON())

		reply := completion.Choices[0].Message.Content
		think := completion.Choices[0].Message.JSON.ExtraFields["reasoning_content"].Raw()

		rsp = map[string]interface{}{
			"query":         query,
			"reply":         reply,
			"think":         think,
			"model_used":    engine.ModelId,                  // 记录实际使用的模型
			"provider_used": engine.GetCurrentProviderName(), // 记录使用的提供商
			"attempts":      attempt,                         // 记录尝试次数
		}
		return rsp, nil
	}

	// 理论上不会到达这里，但为了安全起见
	return nil, fmt.Errorf("未知错误: 所有尝试均未成功：%+v", err)
}

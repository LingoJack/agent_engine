package agent

import (
	"context"
	"encoding/json"
	"log"

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

	// openai-go sdk
	client := openai.NewClient(option.WithAPIKey(engine.GetApiKey()), option.WithBaseURL(engine.BaseUrl))
	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage(query)},
		Model:    engine.ModelId,
	})
	if err != nil {
		return
	}

	log.Printf("raw json: %s", completion.RawJSON())

	reply := completion.Choices[0].Message.Content
	think := completion.Choices[0].Message.JSON.ExtraFields["reasoning_content"].Raw()

	rsp = map[string]interface{}{
		"query": query,
		"reply": reply,
		"think": think,
	}
	return
}

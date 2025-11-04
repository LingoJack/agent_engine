package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"main/agent"
	"main/constant"
	"os"
	"strings"

	markdown "github.com/MichaelMure/go-term-markdown"
	flag "github.com/spf13/pflag"
	"github.com/tidwall/gjson"
	"golang.org/x/term"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

type Response struct {
	Code    int    `json:"code"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

func main() {

	// 日志文件
	logFile, err := os.OpenFile("./agent_engine_logs/log.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		transportResponse(constant.InternalError, nil, "打开日志文件失败")
		return
	}
	writer := io.MultiWriter(logFile)
	log.SetOutput(writer)

	// 从命令行参数中获取参数
	command := flag.StringP("command", "c", "query", "命令, 可选值: query")
	configPath := flag.StringP("conf", "f", "./conf.yaml", "配置文件路径")
	extra := flag.StringP("extract", "e", "$", "提取 JSON PATH 中的某个 key 的 value，例如：$.data.reply")
	modelId := flag.StringP("model", "m", "", "模型名称，如果不指定则选用配置文件中的第一个模型")
	params := flag.StringP("params", "p", "", "参数，string类型")
	providerName := flag.String("provider", "", "提供商名称，如果不指定则选用配置文件中的第一个提供商")

	flag.Parse()

	// 从配置文件创建 Engine
	engine, err := agent.NewEngineFromConfig(*configPath, *providerName, *modelId)
	if err != nil {
		log.Printf("从配置文件创建 Engine 失败: %v", err)
		transportResponse(constant.InternalError, nil, "从配置文件创建 Engine 失败: "+err.Error())
		return
	}
	log.Printf("从配置文件加载: provider=%s, model=%s, baseUrl=%s", engine.GetCurrentProviderName(), engine.ModelId, engine.BaseUrl)

	// db连接
	openedDb, err := gorm.Open(sqlite.Open("./database/agent_db.db"), &gorm.Config{})
	if err != nil {
		transportResponse(constant.InternalError, nil, "数据库连接失败")
		err = nil
	}
	db = openedDb

	// 分发处理，根据结果返回
	data, match, err := engine.DispatchAndHandle(context.Background(), *params, *command)
	if err != nil {
		// 如果没有匹配到事件，返回错误
		if !match {
			transportResponse(constant.EventNotFound, nil, "未找到对应事件")
			return
		}
		// 发生错误
		transportResponse(constant.InternalError, nil, "内部错误: "+err.Error())
		return
	}

	// 如果指定了 extra 参数且不是默认值 "$"，则提取指定路径的值
	if *extra != "" && *extra != "$" {
		// 构建完整的响应结构
		fullResponse := map[string]any{
			"code":    constant.Success,
			"data":    data,
			"message": "success",
		}
		// 将完整响应转换为 JSON 字符串
		jsonData, err := json.Marshal(fullResponse)
		if err != nil {
			log.Printf("序列化数据失败: %v", err)
			transportResponse(constant.InternalError, nil, "序列化数据失败: "+err.Error())
			return
		}

		// 处理 JSONPath 语法：去掉开头的 "$." 前缀（gjson 不需要 $ 前缀）
		extractPath := *extra
		if strings.HasPrefix(extractPath, "$.") {
			extractPath = strings.TrimPrefix(extractPath, "$.")
		}

		// 使用 gjson 提取指定路径的值
		result := gjson.GetBytes(jsonData, extractPath)
		if !result.Exists() {
			log.Printf("提取路径 %s 不存在（原始路径: %s）", extractPath, *extra)
			transportResponse(constant.InternalError, nil, "提取路径不存在: "+*extra)
			return
		}
		// 直接输出提取的值（不包装在响应结构中）
		transport(result.Value(), false)
		return
	}

	// 正常，返回完整数据
	transportResponse(constant.Success, data, "success")
	return
}

// getTerminalWidth 获取终端宽度，如果无法获取则返回默认值
func getTerminalWidth() int {
	// 尝试获取终端宽度
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// 如果获取失败（例如输出被重定向），返回默认宽度
		log.Printf("无法获取终端宽度，使用默认值80: %v", err)
		return 80
	}
	// 确保宽度在合理范围内
	if width < 40 {
		return 40 // 最小宽度
	}
	if width > 200 {
		return 200 // 最大宽度，避免过宽
	}
	return width
}

// transportResponse 返回数据到stdio
func transportResponse(code int, data any, message string) {
	rsp := Response{
		Code:    code,
		Data:    data,
		Message: message,
	}
	transport(rsp, true)
}

// transport 返回数据到stdio，使用自适应的终端宽度和缩进进行markdown渲染
func transport(rsp any, inJson bool) {
	// JSON 格式输出
	if inJson {
		err := json.NewEncoder(os.Stdout).Encode(rsp)
		if err != nil {
			panic(err)
		}
		return
	}

	// Markdown 格式输出：需要先将 any 类型安全地转换为字符串
	var content string
	switch v := rsp.(type) {
	case string:
		// 已经是字符串，直接使用
		content = v
	case []byte:
		// 字节数组，转换为字符串
		content = string(v)
	default:
		// 其他类型，序列化为格式化的 JSON 字符串
		jsonBytes, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			// 序列化失败，使用 fmt.Sprintf 作为后备方案
			content = fmt.Sprintf("%v", v)
		} else {
			content = string(jsonBytes)
		}
	}

	// 获取终端宽度并计算自适应参数
	width := getTerminalWidth()
	// 缩进根据宽度自适应：宽度越大，缩进越大，但保持在合理范围内
	indent := width / 20 // 例如：80列->4空格，100列->5空格，120列->6空格
	if indent < 2 {
		indent = 2 // 最小缩进
	}
	if indent > 8 {
		indent = 8 // 最大缩进
	}

	// 使用自适应参数渲染 markdown
	result := markdown.Render(content, width, indent)
	fmt.Print(string(result))
}

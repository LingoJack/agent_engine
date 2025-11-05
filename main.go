package main

import (
	"agent_engine/agent"
	"agent_engine/constant"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	markdown "github.com/MichaelMure/go-term-markdown"
	flag "github.com/spf13/pflag"
	"github.com/tidwall/gjson"
	"golang.org/x/term"
)

const (
	// DefaultTerminalWidth 终端宽度相关常量
	DefaultTerminalWidth = 80  // 默认终端宽度
	MinTerminalWidth     = 40  // 最小终端宽度
	MaxTerminalWidth     = 200 // 最大终端宽度
	IndentDivisor        = 20  // 缩进计算除数（宽度/20）
	MinIndent            = 2   // 最小缩进
	MaxIndent            = 8   // 最大缩进
)

// Response 定义标准响应结构
type Response struct {
	Code    int    `json:"code"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

func main() {

	// 日志文件
	logFile, err := os.OpenFile("./agent_engine_logs/log.txt", os.O_CREATE|os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		transportResponse(constant.InternalError, nil, "打开日志文件失败")
		return
	}
	writer := io.MultiWriter(logFile)
	log.SetOutput(writer)

	// 自定义 Usage 函数，在 pflag 自动生成的帮助信息前添加简短说明和示例
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Agent Engine - AI 代理引擎命令行工具\n\n")
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n\n", os.Args[0])

		// 命令说明
		fmt.Fprintf(os.Stderr, "命令说明:\n")
		fmt.Fprintf(os.Stderr, "  query   - 向 AI 模型发送查询请求（支持自动模型轮换）\n")
		fmt.Fprintf(os.Stderr, "  list    - 列出所有可用的提供商和模型信息\n")
		fmt.Fprintf(os.Stderr, "  render  - 将 Markdown 文本渲染为终端友好格式\n\n")

		fmt.Fprintf(os.Stderr, "选项:\n")
		// 打印所有 flag 的帮助信息（pflag 自动生成）
		flag.PrintDefaults()

		// 使用示例
		fmt.Fprintf(os.Stderr, "\n使用示例:\n")
		fmt.Fprintf(os.Stderr, "  # 通过参数查询\n")
		fmt.Fprintf(os.Stderr, "  %s -c query -p \"什么是人工智能？\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # 从标准输入查询\n")
		fmt.Fprintf(os.Stderr, "  echo \"你好\" | %s -c query\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # 列出所有模型\n")
		fmt.Fprintf(os.Stderr, "  %s -c list\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # 渲染 Markdown\n")
		fmt.Fprintf(os.Stderr, "  cat README.md | %s -c render\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # 提取特定字段\n")
		fmt.Fprintf(os.Stderr, "  %s -c query -p \"你好\" -e \"$.data.reply\"\n\n", os.Args[0])
	}

	// 定义命令行参数，使用更详细的描述信息（pflag 会自动格式化）
	command := flag.StringP("command", "c", "query",
		"命令类型: query(查询AI), list(列出模型), render(渲染Markdown)")

	configPath := flag.StringP("conf", "f", "./conf.yaml",
		"配置文件路径（支持相对路径和绝对路径）")

	extra := flag.StringP("extract", "e", "$",
		"提取 JSON 响应中指定路径的值，使用 JSONPath 语法（如: $.data.reply）")

	modelId := flag.StringP("model", "m", "",
		"指定使用的模型名称（不指定则使用配置文件中的第一个模型）")

	params := flag.StringP("params", "p", "",
		"命令参数内容（不指定则从标准输入读取；list命令可选，其他命令必需）")

	providerName := flag.String("provider", "",
		"指定提供商名称（不指定则使用配置文件中的第一个提供商）")

	// 添加 help 标志
	help := flag.BoolP("help", "h", false, "显示此帮助信息")

	flag.Parse()

	// 如果用户请求帮助信息，显示后退出
	if *help {
		flag.Usage()
		return
	}

	// 统一处理参数：如果 -p 参数为空，则从标准输入读取
	var inputContent string
	if *params == "" {
		// 从标准输入读取所有内容
		inputBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Printf("从标准输入读取失败: %v", err)
			transportResponse(constant.InternalError, nil, "从标准输入读取失败: "+err.Error())
			return
		}
		inputContent = string(inputBytes)
		// 如果标准输入为空，根据命令类型决定是否报错
		if strings.TrimSpace(inputContent) == "" {
			// list 命令可以不需要参数，其他命令需要内容
			if *command != "list" {
				log.Printf("%s 命令需要内容：请通过 -p 参数指定或从标准输入提供", *command)
				transportResponse(constant.InternalError, nil, fmt.Sprintf("%s 命令需要内容：请通过 -p 参数指定或从标准输入提供", *command))
				return
			}
		}
	} else {
		// 使用命令行参数提供的内容
		inputContent = *params
	}

	// render 命令不需要加载配置文件，直接渲染输出
	if *command == "render" {
		// 直接使用 markdown 渲染并输出
		transport(inputContent, false)
		return
	}

	// 从配置文件创建 Engine
	engine, err := agent.NewEngineFromConfig(*configPath, *providerName, *modelId)
	if err != nil {
		log.Printf("从配置文件创建 Engine 失败: %v", err)
		transportResponse(constant.InternalError, nil, "从配置文件创建 Engine 失败: "+err.Error())
		return
	}
	log.Printf("从配置文件加载: provider=%s, model=%s, baseUrl=%s", engine.GetCurrentProviderName(), engine.ModelId, engine.BaseUrl)

	// 分发处理，根据结果返回（使用统一处理后的 inputContent）
	data, match, err := engine.DispatchAndHandle(context.Background(), inputContent, *command)
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

	if *command == "query" {
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
		log.Printf("无法获取终端宽度，使用默认值%d: %v", DefaultTerminalWidth, err)
		return DefaultTerminalWidth
	}
	// 确保宽度在合理范围内
	if width < MinTerminalWidth {
		return MinTerminalWidth
	}
	if width > MaxTerminalWidth {
		return MaxTerminalWidth
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
	indent := width / IndentDivisor // 例如：80列->4空格，100列->5空格，120列->6空格
	if indent < MinIndent {
		indent = MinIndent
	}
	if indent > MaxIndent {
		indent = MaxIndent
	}

	// 使用自适应参数渲染 markdown
	result := markdown.Render(content, width, indent)
	fmt.Print(string(result))
}

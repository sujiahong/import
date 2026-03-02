package main

import (
	"fmt"
	"log"
	"os"

	"deepseek"
)

func main() {
	// 设置API密钥（实际使用时应该从环境变量获取）
	os.Setenv("DEEPSEEK_API_KEY", "your-api-key-here")

	// 示例1：获取模型列表
	fmt.Println("=== 获取模型列表 ===")
	models, err := deepseek.DSList()
	if err != nil {
		log.Printf("获取模型列表失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个模型\n", len(models.Data))
		for _, model := range models.Data {
			fmt.Printf("  - %s (拥有者: %s)\n", model.ID, model.OwnedBy)
		}
	}

	// 示例2：发送聊天请求
	fmt.Println("\n=== 发送聊天请求 ===")
	question := "你好，请介绍一下自己"
	response, err := deepseek.DSChat(question)
	if err != nil {
		log.Printf("聊天请求失败: %v\n", err)
	} else {
		if len(response.Choices) > 0 {
			fmt.Printf("问题: %s\n", question)
			fmt.Printf("回答: %s\n", response.Choices[0].Message.Content)
			fmt.Printf("使用的模型: %s\n", response.Model)
			fmt.Printf("Token使用: 输入=%d, 输出=%d, 总计=%d\n",
				response.Usage.PromptTokens,
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens)
		}
	}
}

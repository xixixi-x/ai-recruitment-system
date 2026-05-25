package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

type Assistant struct {
	model *openai.ChatModel
	ready bool
}

func New(ctx context.Context, apiKey, baseURL, modelName string) (*Assistant, error) {
	if apiKey == "" {
		return &Assistant{ready: false}, nil
	}
	m, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, err
	}
	return &Assistant{model: m, ready: true}, nil
}

func (a *Assistant) Answer(ctx context.Context, question, businessContext string) (string, error) {
	sys := strings.TrimSpace(`你是智能招聘系统中的 HR 助手。你只能基于系统提供的 MySQL 真实业务统计上下文回答，不做向量检索、不做 RAG、不编造不存在的候选人或岗位。回答要简洁、结构化，并在不确定时说明需要 HR 补充筛选条件。`)

	if !a.ready {
		return fmt.Sprintf("已读取当前系统业务数据，但尚未配置 AI_API_KEY，因此返回规则化摘要。\n\n【业务数据】\n%s\n\n【你的问题】%s\n\n你可以配置 AI_API_KEY、AI_BASE_URL、AI_MODEL 后，由 Eino ChatModel 生成自然语言问答结果。", businessContext, question), nil
	}

	resp, err := a.model.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: sys},
		{Role: schema.User, Content: fmt.Sprintf("以下是从 MySQL 查询得到的真实业务上下文：\n%s\n\nHR 的问题：%s", businessContext, question)},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

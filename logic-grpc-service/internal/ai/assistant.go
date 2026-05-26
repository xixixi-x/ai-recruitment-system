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
	sys := strings.TrimSpace(`你是智能招聘系统中的 HR 数据分析助手。系统会提供从 MySQL 查询得到的真实岗位、投递、候选人档案明细和统计上下文。
你可以基于这些真实数据做计数、排序、比较、归纳和趋势总结，例如分析岗位要求中的技能频次、候选人最高学历、各岗位投递情况、候选人技能分布等。
不要编造上下文中不存在的岗位、候选人、简历内容或数据库字段；如果上下文缺少必要信息，要明确说明缺少什么。
回答要简洁、结构化，优先直接给结论，再给依据。`)

	if !a.ready {
		return fmt.Sprintf("已读取当前系统真实业务明细和统计数据，但尚未配置 AI_API_KEY，因此返回规则化摘要。\n\n【业务数据】\n%s\n\n【你的问题】%s\n\n配置 AI_API_KEY、AI_BASE_URL、AI_MODEL 后，Eino ChatModel 会基于这些业务明细生成自然语言分析结果。", businessContext, question), nil
	}

	resp, err := a.model.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: sys},
		{Role: schema.User, Content: fmt.Sprintf("以下是从 MySQL 查询得到的真实业务明细和统计上下文，请只基于这些数据回答：\n%s\n\nHR 的问题：%s", businessContext, question)},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

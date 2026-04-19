package aihelper

import (
	"context"
	"fmt"

	"GopherAI/model"

	"github.com/cloudwego/eino/schema"
)

type benchmarkNoopModel struct{}

func (benchmarkNoopModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return nil, fmt.Errorf("benchmark helper does not support GenerateResponse")
}

func (benchmarkNoopModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	return "", fmt.Errorf("benchmark helper does not support StreamResponse")
}

func (benchmarkNoopModel) GetModelType() string {
	return "benchmark"
}

// NewBenchmarkAIHelper creates an in-memory helper used by benchmark seed data.
// It is safe to read chat history or accidentally hit chat endpoints without
// dereferencing nil model/save callbacks.
func NewBenchmarkAIHelper(sessionID string) *AIHelper {
	return &AIHelper{
		model:    benchmarkNoopModel{},
		messages: make([]*model.Message, 0),
		SessionID: sessionID,
		saveFunc: func(msg *model.Message) (*model.Message, error) {
			return msg, nil
		},
	}
}

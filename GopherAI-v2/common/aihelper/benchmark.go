package aihelper

import (
	"context"
	"fmt"

	"GopherAI/model"

	"github.com/cloudwego/eino/schema"
)

// benchmarkNoopModel 是一个仅供压测/基线场景使用的“空模型”实现。
// 它不会真正调用任何外部大模型，所有方法都会直接返回错误，
// 用于安全地构造 AIHelper 内存实例，避免误触真实模型调用。
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

// NewBenchmarkAIHelper 构造一个仅用于压测预热的 AIHelper。
// 它不依赖工厂、不依赖外部模型，也不会触发 RabbitMQ 投递，
// 因此可以在 BENCH_MODE 下安全地批量造数。
func NewBenchmarkAIHelper(sessionID string) *AIHelper {
	return &AIHelper{
		model:     benchmarkNoopModel{},
		messages:  make([]*model.Message, 0),
		SessionID: sessionID,
		saveFunc: func(msg *model.Message) (*model.Message, error) {
			return msg, nil
		},
	}
}

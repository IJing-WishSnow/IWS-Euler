package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourname/OrderService/model"
)

// OrderService 订单服务
type OrderService struct {
	orderProducer *Producer // 发送订单到撮合引擎
}

func NewOrderService() *OrderService {
	return &OrderService{
		orderProducer: NewProducer(OrdersTopic),
	}
}

// PlaceOrder 下单：验证 → 发到 Kafka orders topic
func (s *OrderService) PlaceOrder(ctx context.Context, req model.OrderRequest) error {
	// 基本验证
	if req.Quantity <= 0 {
		return fmt.Errorf("数量必须大于 0")
	}
	if req.Type == model.Limit && req.Price <= 0 {
		return fmt.Errorf("限价单价格必须大于 0")
	}
	if req.Symbol == "" {
		return fmt.Errorf("交易对不能为空")
	}

	// 设置时间戳
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}

	// 发送到 Kafka，key 用 symbol 保证同一交易对的订单进同一分区（顺序保证）
	err := s.orderProducer.Send(ctx, req.Symbol, req)
	if err != nil {
		return fmt.Errorf("发送订单到 Kafka 失败: %w", err)
	}

	log.Printf("订单已发送: id=%s side=%d price=%d qty=%d", req.ID, req.Side, req.Price, req.Quantity)
	return nil
}


func (s *OrderService) Close() {
	s.orderProducer.Close()
}

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yourname/IWS-OrderService/model"
)

// 集成测试：需要 Kafka 在 localhost:9094 运行
func TestPlaceOrderToKafka(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	topic := OrdersTopic
	partition := 0

	// 直接连接到指定分区
	conn, err := kafka.DialLeader(ctx, "tcp", KafkaBroker, topic, partition)
	if err != nil {
		t.Fatalf("连接 Kafka 失败: %v", err)
	}
	defer conn.Close()

	// 构造订单
	req := model.OrderRequest{
		ID:        "test-order-1",
		UserID:    "user-1",
		Symbol:    "BTC/USDT",
		Side:      model.Buy,
		Type:      model.Limit,
		Price:     50000,
		Quantity:  10,
		CreatedAt: time.Now(),
	}
	data, _ := json.Marshal(req)

	// 写入消息到分区 0
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.WriteMessages(kafka.Message{Value: data})
	if err != nil {
		t.Fatalf("写入消息失败: %v", err)
	}
	t.Log("订单已写入 Kafka partition 0")

	// 用新连接读取（从最新 offset 往前一条）
	readConn, err := kafka.DialLeader(ctx, "tcp", KafkaBroker, topic, partition)
	if err != nil {
		t.Fatalf("读取连接失败: %v", err)
	}
	defer readConn.Close()

	// 获取最新 offset，往前退一条
	lastOffset, _ := readConn.ReadLastOffset()
	readConn.Seek(lastOffset-1, kafka.SeekAbsolute)
	readConn.SetReadDeadline(time.Now().Add(5 * time.Second))

	batch := readConn.ReadBatch(1, 1e6)
	defer batch.Close()

	msg := make([]byte, 1e6)
	n, err := batch.Read(msg)
	if err != nil {
		t.Fatalf("读取消息失败: %v", err)
	}

	var got model.OrderRequest
	if err := json.Unmarshal(msg[:n], &got); err != nil {
		t.Fatalf("解析消息失败: %v", err)
	}

	if got.ID != "test-order-1" {
		t.Fatalf("期望订单ID test-order-1，实际 %s", got.ID)
	}
	if got.Price != 50000 {
		t.Fatalf("期望价格 50000，实际 %d", got.Price)
	}

	t.Logf("Kafka 消息收发验证通过: id=%s price=%d qty=%d", got.ID, got.Price, got.Quantity)
}

// 测试 OrderService.PlaceOrder 方法的验证逻辑
func TestPlaceOrderValidation(t *testing.T) {
	ctx := context.Background()
	svc := NewOrderService()
	defer svc.Close()

	tests := []struct {
		name    string
		req     model.OrderRequest
		wantErr bool
	}{
		{"数量为0", model.OrderRequest{Symbol: "BTC/USDT", Type: model.Limit, Price: 100, Quantity: 0}, true},
		{"限价单价格为0", model.OrderRequest{Symbol: "BTC/USDT", Type: model.Limit, Price: 0, Quantity: 10}, true},
		{"交易对为空", model.OrderRequest{Type: model.Limit, Price: 100, Quantity: 10}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.PlaceOrder(ctx, tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("期望报错，实际没有")
			}
		})
	}
}

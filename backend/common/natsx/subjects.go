// Package natsx 封装 NATS JetStream 客户端(连接、stream 与 durable consumer 管理)。
// 流/主题命名必须与 backend/ingestion/subscribe.py 保持同步,不可单方面修改。
package natsx

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamTick     string = "tick_stream" // 原始 tick 流,订阅 tick.>
	StreamBar      string = "bar_stream"  // 原始 bar 流,订阅 bar.>
	SubjectTickNew string = "tick.new"    // 每条 tick
	SubjectBarNew  string = "bar.new"     // 每个完成的 bar
)

// 注意:stream 配置必须持久保留(retention/存储上限按磁盘规划),
// 消费侧一律用 durable consumer + AckExplicit,崩溃重启后从 ack 位点续传。
// 禁止启动时 DeleteStream 重建(旧 cmd/server 的写法会丢数据)。

type NatsxClient struct {
	conn *nats.Conn
}

func NewNatsxClient(url string) (*NatsxClient, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("fail to connect to nats: %w", err)
	}
	return &NatsxClient{conn: nc}, nil
}

func (nc *NatsxClient) Close() {
	if nc != nil && nc.conn != nil {
		nc.conn.Close()
	}
}

func (nc *NatsxClient) Flush() {
	if nc != nil && nc.conn != nil {
		nc.conn.Flush()
	}
}

type JetStreamClient struct {
	natsClient *NatsxClient
	js         jetstream.JetStream
}

func NewJetStreamClient(url string) (*JetStreamClient, error) {
	nc, err := NewNatsxClient(url)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc.conn)
	if err != nil {
		nc.Close() // 创建 JS 失败时及时释放 NATS 连接
		return nil, fmt.Errorf("fail to create jetstream context: %w", err)
	}

	return &JetStreamClient{
		natsClient: nc,
		js:         js,
	}, nil
}

func (jsc *JetStreamClient) Close() {
	if jsc != nil && jsc.natsClient != nil {
		jsc.natsClient.Close()
	}
}

func (jsc *JetStreamClient) Flush() {
	if jsc != nil && jsc.natsClient != nil {
		jsc.natsClient.Flush()
	}
}

func (jsc *JetStreamClient) GetStream(ctx context.Context, streamName string) (jetstream.Stream, error) {
	return jsc.js.Stream(ctx, streamName)
}

func (jsc *JetStreamClient) CreateStream(ctx context.Context, streamName string, subjects []string) (jetstream.Stream, error) {
	// 复用外层 ctx
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return jsc.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: subjects,
	})
}

// 显示将 ctx 作为第一个参数传入，并挂载超时控制
func (jsc *JetStreamClient) CreateDurableConsumer(ctx context.Context, stream jetstream.Stream, durableName string) (jetstream.Consumer, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   durableName,
		AckPolicy: jetstream.AckExplicitPolicy,
	})
}

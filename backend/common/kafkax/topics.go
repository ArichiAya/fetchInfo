// Package kafkax 封装 Kafka 客户端,集中定义主题名和固定配置。
// 客户端实现待接入(候选:franz-go / segmentio/kafka-go)。
package kafkax

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

// 消息 key 统一为 symbol,保证同一标的的消息在单分区内有序;
// 不同下游使用不同 consumer group,各自独立伸缩。
const (
	TopicTick               = "tick2.new"
	TopicBar                = "bar2.new"
	PartitionAmount   int32 = 3
	ReplicationAmount int16 = 1
)

type KafkaxAdmin struct {
	cl  *kgo.Client
	adm *kadm.Client
}

func NewKafkaxAdmin(opts ...kgo.Opt) (*KafkaxAdmin, error) {
	// 初始化franz-go客户端
	kcl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// 基于 kgo.Client 创建 kadm 管理客户端
	kadm := kadm.NewClient(kcl)

	kxadm := &KafkaxAdmin{
		cl:  kcl,
		adm: kadm,
	}

	return kxadm, nil
}

func (kxadm *KafkaxAdmin) Close() {
	if kxadm != nil && kxadm.cl != nil {
		kxadm.cl.Close()
	}
}

func (kxadm *KafkaxAdmin) Flush(ctx context.Context) {
	if kxadm != nil && kxadm.cl != nil {
		kxadm.cl.Flush(ctx)
	}
}

// CheckTopics 返回不存在的 Topic 切片,并返回无法加载的 Topic(if exit)的err信息
func (kxadm *KafkaxAdmin) CheckTopics(ctx context.Context, targetTopics []string) (*[]string, *[]error) {
	if kxadm == nil || kxadm.adm == nil {
		return nil, &[]error{fmt.Errorf("nil ptr error: kxadm is nil or its init was failed")}
	}

	topicDetials, err := kxadm.adm.ListTopics(ctx, targetTopics...)
	if err != nil {
		return nil, &[]error{fmt.Errorf("Fail to list topics: %w", err)}
	}

	errs := []error{}
	noExitTopics := make([]string, 0, len(targetTopics))
	for _, targetTopic := range targetTopics {
		if details, exists := topicDetials[targetTopic]; exists {
			if details.Err != nil {
				errs = append(errs, fmt.Errorf("Fail to load Topic %s cause: %w\n", details.Topic, details.Err))
			}
			if details.Err == kerr.UnknownTopicOrPartition {
				noExitTopics = append(noExitTopics, targetTopic)
			}
		}
	}

	return &noExitTopics, &errs
}

// CreateTopics 批量创建 Topic
func (kxadm *KafkaxAdmin) CreateTopics(ctx context.Context, configs map[string]*string, topics []string) error {
	if kxadm == nil || kxadm.adm == nil {
		return fmt.Errorf("nil ptr error: kxadm is nil or its init was failed")
	}

	resps, err := kxadm.adm.CreateTopics(ctx, PartitionAmount, ReplicationAmount, configs, topics...)
	if err != nil {
		return fmt.Errorf("failed to issue create topic request: %w", err)
	}

	for _, resp := range resps {
		if resp.Err != nil {
			return fmt.Errorf("failed to create topic %s: %v(cause: %s)", resp.Topic, resp.Err, resp.ErrMessage)
		}
	}

	return nil
}

func (kxadm *KafkaxAdmin) Produce(ctx context.Context, record *kgo.Record, promise func(*kgo.Record, error)) {
	if kxadm != nil && kxadm.cl != nil {
		// 异步发送（Produce）配合回调函数，适用于追求极致吞吐量的场景
		kxadm.cl.Produce(ctx, record, promise)
	}
}

// PollFetches返回kgo.Fetches. timeout>0时传入携带DeadLine的Context,否则就透传
func (kxadm *KafkaxAdmin) PollFetches(ctx context.Context, timeout time.Duration) kgo.Fetches {
	if timeout > 0*time.Second {
		timeoutctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return kxadm.cl.PollFetches(timeoutctx)
	}
	return kxadm.cl.PollFetches(ctx)
}

func (kxadm *KafkaxAdmin) Commit(ctx context.Context) error {
	uncommitted := kxadm.cl.UncommittedOffsets()
	for topic, partitions := range uncommitted {
		for partition, epochOffset := range partitions {
			fmt.Printf(" [待提交] Topic: %s | Partition: %d | Next Offset: %d\n",
				topic, partition, epochOffset.Offset)
		}
	}
	err := kxadm.cl.CommitUncommittedOffsets(ctx)

	committed := kxadm.cl.CommittedOffsets()
	for topic, partitions := range committed {
		for partition, epochOffset := range partitions {
			fmt.Printf(" [已提交] Topic: %s | Partition: %d | Current Offset: %d\n",
				topic, partition, epochOffset.Offset)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to commit offsets: %w", err)
	}
	return nil
}

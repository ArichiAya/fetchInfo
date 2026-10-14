// Command compute:NATS消费 → 计算指标 → Kafka。
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ArichiAya/fetch-info/common/kafkax"
	"github.com/ArichiAya/fetch-info/common/msg"
	"github.com/ArichiAya/fetch-info/common/natsx"
	"github.com/ArichiAya/fetch-info/market/internal/indicator"
	"github.com/ArichiAya/fetch-info/market/internal/model"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"
)

var (
	natsServerURL string
	kafkaBroker1  string
)

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("load .env error: %v", err)
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}
	natsServerURL = os.Getenv("natsServerURL")
	kafkaBroker1 = os.Getenv("kafkaBroker1")
}

func main() {
	// 设置日志输出到文件和标准输出
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// 组合文件和标准输出
	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)

	// 初始化jsc
	jsctx := context.Background()
	jsc, err := natsx.NewJetStreamClient(natsServerURL)
	if err != nil {
		log.Fatalf("Fail to create the JetStreamClient: %v", err)
	}
	defer jsc.Close()

	// 获取TickStream,不存在则创建
	tickNewStream, err := jsc.GetStream(jsctx, natsx.StreamTick)
	if err != nil {
		// 创建TickStream
		tickNewStream, err = jsc.CreateStream(jsctx, natsx.StreamTick, []string{natsx.SubjectTickNew})
		if err != nil {
			log.Fatalf("Fail to create the tick new stream: %v", err)
		}
	}

	// 创建一个TickDurableConsumer
	tickDurableConsumerName := "TICKCONS"
	tickCons, err := jsc.CreateDurableConsumer(jsctx, tickNewStream, tickDurableConsumerName)
	if err != nil {
		log.Fatalf("Fail to create the tick consumer: %v", err)
	}

	// TODO: 创建BarDurableCOnsumer

	// 初始化KafkaxAdmin
	bkgctx := context.Background()
	kgoopts := []kgo.Opt{
		kgo.SeedBrokers(kafkaBroker1),
	}

	kxadm, err := kafkax.NewKafkaxAdmin(kgoopts...)
	if err != nil {
		log.Fatalf("func NewKafkaxAdmin err,cause: %v", err)
	}
	defer kxadm.Close()

	// 检查某个 Topic 是否存在
	targetTopics := []string{kafkax.TopicTick, kafkax.TopicBar}
	admctx4Check, cancel4Check := context.WithTimeout(bkgctx, 5*time.Second)
	defer cancel4Check()

	noExitTopics, errs := kxadm.CheckTopics(admctx4Check, targetTopics)
	if len(*errs) != 0 {
		log.Printf("CheckTopics error: %v", *errs)
	}

	// 创建 Topic (if not exit)
	configs := map[string]*string{
		"cleanup.policy": kadm.StringPtr("delete"),   // 可选配置：如 delete 或 compact
		"retention.ms":   kadm.StringPtr("86400000"), // 可选配置：保留 1 天 (24小时)
	}

	admctx4Create, cancel4Create := context.WithTimeout(bkgctx, 30*time.Second)
	defer cancel4Create()
	log.Printf("Creating topics: %v", *noExitTopics)
	err = kxadm.CreateTopics(admctx4Create, configs, *noExitTopics)
	if err != nil {
		log.Fatalf("Fail to create topics: %v", err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	tickcounter := 0

	go func() {
		for {
			select {
			case <-done:
				jsc.Flush()
				kxadm.Flush(bkgctx)
				log.Println("Cleaning up...")
				log.Printf("Total messages processed: %d", tickcounter)
				wg.Done()
				return
			default:
				// Pull 3 messages from the consumer
				messages, err := tickCons.Fetch(3, jetstream.FetchMaxWait(10*time.Second))
				if err != nil {
					log.Fatalf("Fail to fetch the messages: %s", err)
				}
				for message := range messages.Messages() {

					msgData := message.Data()
					newtick := &msg.Tick{}
					if err := proto.Unmarshal(msgData, newtick); err != nil {
						log.Fatalf("Fail to Unmarshal Tick message: %v", err)
					}
					// 打印tick信息
					// log.Printf("Received tick for %s: %s", newtick.Symbol, newtick.String())
					// 计算指标
					iAVG := indicator.CaliAVG(newtick.Quotes)
					quoteslength := float64(len(newtick.Quotes))
					newt2 := model.NewTick2(
						model.WithTickRow(newtick),
						model.WithDelta_V(indicator.CalDeltaV(newtick.Price, newtick.Quotes[0].AskP, newtick.Quotes[0].BidP, newtick.LastVolume)),
						model.WithVWAP(indicator.CalVWAP(newtick.CumAmount, float64(newtick.CumVolume))),
						model.WithP_Book(indicator.CalPBook(newtick.Quotes)),
						model.WithSlope_Bid(indicator.CalSlopeBid(newtick.Quotes, iAVG, indicator.CalbidvAVG(newtick.Quotes, quoteslength))),
						model.WithSlope_Ask(indicator.CalSlopeAsk(newtick.Quotes, iAVG, indicator.CalaskvAVG(newtick.Quotes, quoteslength))),
						model.WithSpread(indicator.CalSpread(newtick.Quotes[0].AskP, newtick.Quotes[0].BidP)),
					)
					t2message, err := proto.Marshal(newt2)
					if err != nil {
						log.Fatalf("Fail to Marshal Tick2 message: %v", err)
					}

					// 构造 Record
					record := &kgo.Record{
						Key:   []byte(newt2.TickRow.Symbol),
						Value: t2message,
						Topic: kafkax.TopicTick,
					}

					// 异步发送（Produce）配合回调函数
					kxadm.Produce(bkgctx, record, func(r *kgo.Record, err error) {
						if err != nil {
							fmt.Printf("Fail to send the message to kafka broker: %v\n", err)
							return
						}
						fmt.Printf("Success sending the message to kafka broker,Topic: %s, Partition: %d, Offset: %d\n", r.Topic, r.Partition, r.Offset)
						message.Ack()
						tickcounter++
					})

				}
			}
		}
	}()

	<-shutdown
	close(done)
	wg.Wait()
	log.Println("[compute] shutdown")
}

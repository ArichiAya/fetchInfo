// Command persister:行情持久化服务。
//
// 数据流:
//
//	Kafka(tick2.new / bar2.new,consumer group "timescale-writer")
//	  → internal/persist:批量写入 TimescaleDB hypertable
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ArichiAya/fetch-info/common/consts"
	"github.com/ArichiAya/fetch-info/common/kafkax"
	"github.com/ArichiAya/fetch-info/common/msg"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"
)

const (
	kafkaBroker1          string = "192.168.80.131:9092"
	databaseURL           string = "postgres://iiot_admin:changeme_please_2024@192.168.80.131:5432/mktdata"
	consumerGroup         string = "timescale-writer"
	exit                  int    = 0
	faltalError           int    = 1
	commitjob             int    = 2
	tickTableEnd          string = "tick2"
	barTableEnd           string = "bar2"
	tickTableColumeAmount        = 9
	barTableColumeAmount         = 4
)

var topics = []string{kafkax.TopicTick, kafkax.TopicBar}

type (
	recordContainer    chan *kgo.Record
	statusCodeSender   chan<- int
	statusCodeReciever <-chan int
)

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

	// TODO:
	// client fetches获取消息,fetches.EachRecord获取批次里的记录，将记录按Topic区分，随后注册在对应的map里，创建一个goroutine，子goroutine里从channel中批量获取消息
	// 当达到一定数量或者time.TIker传递消息时触发持久化函数，写入到TimeScaleDB里
	// todo: 做幂等处理,防止收到重复消息,原因: publisher可能存在发送给NATS成功但因为网络问题或事件循环异常导致未收到ack,导致publisher重发消息.
	// 例如可以用tick的symbol+created_at和bar的symbol+bob作为唯一标识,让业务本身做幂等处理.

	bkgctx := context.Background()
	shutdown := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup

	// 每个Topic创建一个*record channel
	messages := map[string]recordContainer{}
	for _, topic := range topics {
		messages[topic] = make(recordContainer, 10)
	}

	status := make(chan int, 1)
	// var sender statusCodeSender = status

	// 每个topic启动一个goroutine进行持久化
	for t, c := range messages {
		go persist(t, c, done, status, &wg)
		wg.Add(1)
	}

	// 每个Topic创建一个Consumer client
	for range topics {
		go runConsumerWorker(bkgctx, &messages, done, status, &wg)
		wg.Add(1)
	}

	<-shutdown
	close(done)
	wg.Wait()
	log.Println("[persister] shutdown")
}

func runConsumerWorker(ctx context.Context, messages *map[string]recordContainer, stopSig <-chan struct{}, status chan int, wg *sync.WaitGroup) {
	// 初始化kxadm
	kgoopts := []kgo.Opt{
		kgo.SeedBrokers(kafkaBroker1),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topics...),
		kgo.DisableAutoCommit(), // 关闭隐式自动提交
	}
	kxadm, err := kafkax.NewKafkaxAdmin(kgoopts...)
	if err != nil {
		log.Fatalf("Fail to new a kafkaxAdmmin: %v", err)
	}
	defer kxadm.Close()

	// 为当前topic的客户端启动offset提交goroutine
	go commit(ctx, kxadm, status, wg)
	wg.Add(1)

	// loop logic
	for {
		select {
		case <-stopSig:
			kxadm.Flush(ctx)
			wg.Done()
			return
		default:
			fetches := kxadm.PollFetches(ctx, 5*time.Second)
			if len(fetches) == 0 {
				continue
			}

			fetches.EachError(func(Topic string, Partition int32, Err error) {
				log.Printf("fetch error of %s in Partition %d: %v", Topic, Partition, err)
			})

			fetches.EachRecord(func(record *kgo.Record) {
				// 根据record.Topic将*record发送到对应channel
				(*messages)[record.Topic] <- record
			})

		}
	}
}

// TODO: 完善持久化逻辑
func persist(topic string, rc recordContainer, stopSig <-chan struct{}, status statusCodeSender, wg *sync.WaitGroup) {
	// 初始化pg连接
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Printf("Fail to Create pgx pool :%v", err)
		status <- faltalError
		wg.Done()
		return
	}

	ticker := time.Tick(3 * time.Second)
	batch := &pgx.Batch{}
	var tableEnd string

	if topic == kafkax.TopicTick {
		tableEnd = tickTableEnd
	} else {
		tableEnd = barTableEnd
	}

	for {
		select {
		case <-stopSig:
			// TODO: Flush
			status <- exit
			wg.Done()

		case <-ticker:
			if batch.Len() == 0 {
				continue
			}

			execflag := true
			result := pool.SendBatch(ctx, batch)
			for i := 0; i < batch.Len(); i++ {
				_, err := result.Exec()
				if err != nil {
					execflag = false
					log.Printf("Query exec error: %v", err)
				}
			}
			result.Close()

			if execflag {
				log.Printf("Batch insert %d records successfully", batch.Len())
			}
			status <- commitjob
			batch = &pgx.Batch{}

		case rcd := <-rc:
			msgData, err := getUnmarshaledData(topic, rcd)
			if err != nil {
				log.Printf("Fail to unmarshal message cause: %v", err)
				continue
			}

			err = queueQuery(topic, msgData, tableEnd, batch)
			if err != nil {
				log.Printf("Fail to queue the sql str to the batch: %v", err)
			}

		}
	}
}

func commit(ctx context.Context, kxadm *kafkax.KafkaxAdmin, status statusCodeReciever, wg *sync.WaitGroup) {
	for {
		switch code := <-status; code {
		case commitjob:
			// 提交offset
			if err := kxadm.Commit(ctx); err != nil {
				log.Printf("Failt to commit uncommitted offsets: %v", err)
			} else {
				log.Println("Commit uncommitted offsets successfully")
			}
		case faltalError:
			log.Printf("commit goroutine exit with code: %d", code)
			wg.Done()
			return
		case exit:
			wg.Done()
			log.Printf("Receive exit code %d", code)
			return
		}
	}
}

func valuesParameterHolderString(amount int) string {
	if amount < 1 {
		return ""
	}
	var ret strings.Builder
	ret.WriteString(" (")
	for i := 1; i <= amount; i++ {
		if i > 1 {
			ret.WriteString(",")
		}
		ret.WriteString("$")
		ret.WriteString(strconv.Itoa(i))
	}
	ret.WriteString(") ")
	return ret.String()
}

func getArguments(topic string, objpointer any) (*[]any, error) {
	timeLayout := "2006-01-02 15:04:05.999999"

	switch topic {

	case kafkax.TopicTick:
		value, ok := objpointer.(*msg.Tick2)
		if !ok {
			return nil, fmt.Errorf("Fail to assert tick2 pointer")
		}

		createdAt, err := time.Parse(timeLayout, value.TickRow.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("Fail to Parse: %w", err)
		}

		tickRow, err := json.Marshal(value.TickRow)
		if err != nil {
			return nil, fmt.Errorf("Fail to marshal tickrow to jsonb: %w", err)
		}

		return &[]any{createdAt, value.TickRow.Symbol, tickRow, value.DeltaV, value.Vwap, value.PBook, value.SlopeBid, value.SlopeAsk, value.Spread}, nil

	case kafkax.TopicBar:
		value, ok := objpointer.(*msg.Bar2)
		if !ok {
			return nil, fmt.Errorf("Fail to assert bar2 pointer")
		}

		createdAt, err := time.Parse(timeLayout, value.BarRow.Bob)
		if err != nil {
			return nil, fmt.Errorf("Fail to Parse: %w", err)
		}

		barRow, err := json.Marshal(value.BarRow)
		if err != nil {
			return nil, fmt.Errorf("Fail to marshal barrow to jsonb: %w", err)
		}

		return &[]any{createdAt, value.BarRow.Symbol, barRow, value.BodyRatio}, nil
	}

	return nil, fmt.Errorf("Didn't match any topic when calling getArguments")
}

func getUnmarshaledData(topic string, rcd *kgo.Record) (any, error) {
	switch topic {
	case kafkax.TopicTick:
		ret := &msg.Tick2{}
		err := proto.Unmarshal(rcd.Value, ret)
		return ret, err
	case kafkax.TopicBar:
		ret := &msg.Bar2{}
		err := proto.Unmarshal(rcd.Value, ret)
		return ret, err
	}
	return nil, fmt.Errorf("Didn't match any topic when calling getUnmarshaledData")
}

func queueQuery(topic string, msgData any, tableEnd string, batch *pgx.Batch) error {
	switch topic {
	case kafkax.TopicTick:
		msgTick2, ok := msgData.(*msg.Tick2)
		if !ok {
			return fmt.Errorf("Fail to assert tick2 pointer")
		}
		tableName := fmt.Sprintf("%s_%s", msgTick2.TickRow.Symbol, tableEnd)
		tableName = fixTableName(tableName)
		ags, err := getArguments(topic, msgTick2)
		if err != nil {
			return fmt.Errorf("An error occur when calling getArguments: %w", err)
		}

		columAmount := tickTableColumeAmount
		safeQueryStrPre := fmt.Sprintf("INSERT INTO %s.%s VALUES ", consts.DefaultSchemaName, pgx.Identifier{tableName}.Sanitize())
		safeQueryStr := safeQueryStrPre + valuesParameterHolderString(columAmount) + "ON CONFLICT (created_at) DO NOTHING"
		log.Printf("%s queued", safeQueryStr)
		batch.Queue(safeQueryStr, (*ags)...)

		return nil
	case kafkax.TopicBar:
		msgBar2, ok := msgData.(*msg.Bar2)
		if !ok {
			return fmt.Errorf("Fail to assert bar2 pointer")
		}

		tableName := fmt.Sprintf("%s_%s", msgBar2.BarRow.Symbol, tableEnd)
		tableName = fixTableName(tableName)
		ags, err := getArguments(topic, msgBar2)
		if err != nil {
			return fmt.Errorf("An error occur when calling getArguments: %w", err)
		}

		columAmount := barTableColumeAmount
		safeQueryStrPre := fmt.Sprintf("INSERT INTO %s.%s VALUES ", consts.DefaultSchemaName, pgx.Identifier{tableName}.Sanitize())
		safeQueryStr := safeQueryStrPre + valuesParameterHolderString(columAmount) + "ON CONFLICT (created_at) DO NOTHING"
		batch.Queue(safeQueryStr, (*ags)...)
		return nil
	}
	return fmt.Errorf("Didn't match any topic when calling queueQuery")
}

// fixTableName symbol 里含有 '.' ,转换成 '_';并把大写字母转换成小写字母
func fixTableName(tableName string) string {
	tableName = strings.ReplaceAll(tableName, ".", "_")
	tableName = strings.ToLower(tableName)
	return tableName
}

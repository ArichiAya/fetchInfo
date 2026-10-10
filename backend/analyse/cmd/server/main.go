package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	NATSServerURL string = "nats://192.168.80.131:4222"
	PostgresURL   string = "postgresql://iiot_admin:changeme_please_2024@192.168.80.131:5432/iiot_db"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Connect to a server
	nc, err := nats.Connect(NATSServerURL)
	if err != nil {
		log.Fatalf("[error] Fail to connect to the Nats: %s", err)
	}
	defer nc.Close()

	// create jetstream context from nats connection
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("[error] Fail to Create the jetstream instance: %s", err)
	}

	// Create a stream
	_ = js.DeleteStream(ctx, "tick_stream") // Delete the stream if it already exists
	_ = js.DeleteStream(ctx, "bar_stream")  // Delete the stream if it already exists
	tickStream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "tick_stream",
		Subjects: []string{"tick.>"},
	})
	if err != nil {
		log.Fatalf("[error] Fail to create stream: %s", err)
	}

	// Create durable consumer
	tickConsumer, _ := tickStream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "CONS",
		AckPolicy: jetstream.AckExplicitPolicy,
	})

	done := make(chan struct{})
	counter := 0
	go func() {
		// todo: 做幂等处理,防止收到重复消息,原因: publisher可能存在发送给NATS成功但因为网络问题或事件循环异常导致未收到ack,导致publisher重发消息.
		// 例如可以用tick的symbol+created_at和bar的symbol+bob作为唯一标识,让业务本身做幂等处理.
		for {
			select {
			case <-done:
				log.Println("Cleaning up...")
				log.Printf("Total messages processed: %d", counter)
				return
			default:
				// Pull 3 messages from the consumer
				msgs, err := tickConsumer.Fetch(3, jetstream.FetchMaxWait(10*time.Second))
				if err != nil {
					log.Fatalf("[error] Fail to fetch the messages: %s", err)
				}
				for msg := range msgs.Messages() {
					msg.Ack()
					counter++
					log.Printf("Received a message via fetch: %s\n", string(msg.Data()))
				}
			}
		}
	}()

	log.Println("Server is running. Press Ctrl+C to exit.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down...")

	close(done)
}

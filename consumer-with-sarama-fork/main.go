package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davidandradeduarte/sarama"
)

const (
	broker           = ""
	consumerGroup    = ""
	topic            = ""
	connectionString = ""
)

func main() {

	brokerList := []string{broker}
	fmt.Println("Event Hubs broker", brokerList)
	consumerGroupID := consumerGroup
	fmt.Println("Sarama client consumer group ID", consumerGroupID)

	consumer, err := sarama.NewConsumerGroup(brokerList, consumerGroupID, getConfig())

	if err != nil {
		fmt.Println("error creating new consumer group", err)
		os.Exit(1)
	}

	fmt.Println("new consumer group created")

	eventHubsTopic := topic
	fmt.Println("Event Hubs topic", eventHubsTopic)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			err = consumer.Consume(ctx, []string{eventHubsTopic}, messageHandler{})
			if err != nil {
				fmt.Println("error consuming from group", err)
				os.Exit(1)
			}

			if ctx.Err() != nil {
				return
			}
		}
	}()

	close := make(chan os.Signal)
	signal.Notify(close, syscall.SIGTERM, syscall.SIGINT)
	fmt.Println("Waiting for program to exit")
	<-close
	cancel()
	fmt.Println("closing consumer group....")

	if err := consumer.Close(); err != nil {
		fmt.Println("error trying to close consumer", err)
		os.Exit(1)
	}
	fmt.Println("consumer group closed")
}

type messageHandler struct{}

func (h messageHandler) Setup(s sarama.ConsumerGroupSession) error {
	fmt.Println("Partition allocation -", s.Claims())
	return nil
}

func (h messageHandler) Cleanup(s sarama.ConsumerGroupSession) error {
	fmt.Println("Consumer group clean up initiated")
	return nil
}
func (h messageHandler) ConsumeClaim(s sarama.ConsumerGroupSession, c sarama.ConsumerGroupClaim) error {
	for msg := range c.Messages() {
		fmt.Printf("Message topic:%q partition:%d offset:%d\n", msg.Topic, msg.Partition, msg.Offset)
		fmt.Println("Message content", string(msg.Value))
		s.MarkMessage(msg, "")
	}
	return nil
}

func getConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second

	config.Net.SASL.Enable = true
	config.Net.SASL.User = "$ConnectionString"
	config.Net.SASL.Password = connectionString
	config.Net.SASL.Mechanism = "PLAIN"

	config.Net.TLS.Enable = true
	config.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
		ClientAuth:         0,
	}
	config.Version = sarama.V1_0_0_0

	return config
}

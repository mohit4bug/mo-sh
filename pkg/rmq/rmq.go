package rmq

import (
	"context"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	rmqConn    *amqp.Connection
	rmqChannel *amqp.Channel
	rmqOnce    sync.Once
)

func InitRMQ() {
	rmqOnce.Do(func() {
		var err error
		rmqConn, err = amqp.Dial("amqp://admin:password@localhost:5672/")
		if err != nil {
			log.Fatal(err)
		}

		rmqChannel, err = rmqConn.Channel()
		if err != nil {
			log.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		select {
		case <-ctx.Done():
			log.Fatal("RabbitMQ connection timeout")
		default:
			log.Println("Successfully connected to RabbitMQ")
		}
	})
}

func GetRMQChannel() *amqp.Channel {
	if rmqChannel == nil {
		log.Panic("RabbitMQ channel is not initialized")
	}
	return rmqChannel
}

func GetRMQConnection() *amqp.Connection {
	if rmqConn == nil {
		log.Panic("RabbitMQ connection is not initialized")
	}
	return rmqConn
}

func CloseRMQ() {
	if rmqChannel != nil {
		if err := rmqChannel.Close(); err != nil {
			log.Fatal(err)
		} else {
			log.Println("RabbitMQ channel closed")
		}
	}

	if rmqConn != nil {
		if err := rmqConn.Close(); err != nil {
			log.Fatal(err)
		} else {
			log.Println("RabbitMQ connection closed")
		}
	}
}

package main

import (
	"context"
	sitemap "github.com/oxffaa/gopher-parse-sitemap"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

const amqpProtocol = "amqp"

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func getRabbitMqDSN() string {
	builder := strings.Builder{}
	builder.WriteString(amqpProtocol)
	builder.WriteString("://")
	builder.WriteString(os.Getenv("RABBITMQ_USER"))
	builder.WriteString(":")
	builder.WriteString(os.Getenv("RABBITMQ_PASSWORD"))
	builder.WriteString("@")
	builder.WriteString(os.Getenv("RABBITMQ_URL"))
	builder.WriteString("/")
	builder.WriteString(os.Getenv("RABBITMQ_VIRTUAL_HOST"))
	return builder.String()
}

func main() {
	rand.Seed(time.Now().UnixNano())
	conn, err := amqp.Dial(getRabbitMqDSN())
	defer conn.Close()
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	defer ch.Close()
	failOnError(err, "Failed to open a channel")
	err = ch.ExchangeDeclare(
		"urls.exchange", // name
		"fanout",        // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	failOnError(err, "Failed to open urls.exchange")

	_, err = ch.QueueDeclare(
		"queue.urls", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "death.exchange",
		}, // arguments
	)
	failOnError(err, "Failed to declare a urls queue")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	failOnError(err, "Failed to connect to RabbitMQ")

	err = sitemap.ParseFromFile("sitemap.xml", func(e sitemap.Entry) error {
		body := e.GetLocation()
		err = ch.PublishWithContext(ctx,
			"urls.exchange", // exchange
			"",              // routing key
			//q.Name, // routing key
			false, // mandatory
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "text/plain",
				Body:         []byte(body),
			})
		failOnError(err, "Failed to publish a message")
		log.Printf(" queue-publisher: Url: %s", body)
		return nil
	})
	failOnError(err, "Failed to declare a queue")
}

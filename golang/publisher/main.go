package main

import (
	"context"
	"fmt"
	sitemap "github.com/oxffaa/gopher-parse-sitemap"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"math/rand"
	"time"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Hello, Go!")

	//conn, err := amqp.Dial("amqp://root:dupQuGgxiEedj345hL9D25MzDHCFDweR@queue_rabbitmq:5672/queue_rabbitmq")
	conn, err := amqp.Dial("amqp://root:dupQuGgxiEedj345hL9D25MzDHCFDweR@queue-rabbitmq:5672/queue_rabbitmq")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"urls", // name
		true,   // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	failOnError(err, "Failed to declare a queue")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err != nil {
		panic(err)
	}
	err = sitemap.ParseFromFile("sitemap.xml", func(e sitemap.Entry) error {
		body := e.GetLocation()
		err = ch.PublishWithContext(ctx,
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "text/plain",
				Body:         []byte(body),
			})
		failOnError(err, "Failed to publish a message")
		log.Printf(" [x] Sent %s", body)
		return nil
	})

	failOnError(err, "Failed to declare a queue")
}

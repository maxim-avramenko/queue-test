package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type responseData struct {
	md5hash         string
	url             string
	response_status int
	headers         string
	body            []byte
}

func newResponseData(md5hash string, url string, responseStatus int, headers string, body []byte) *responseData {
	return &responseData{
		md5hash:         md5hash,
		url:             url,
		response_status: responseStatus,
		headers:         headers,
		body:            body,
	}
}

func getMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {

	db, err := sql.Open(
		"mysql",
		os.Getenv("DBUSER")+":"+os.Getenv("DBPASS")+"@"+os.Getenv("DBURL")+"/"+os.Getenv("DBNAME")+"?charset=utf8mb4,utf8")
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	// Capture rabbitmq connection properties.
	rabbitMQconnectString := "amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@" + os.Getenv("RABBITMQ_URL") + "/" + os.Getenv("RABBITMQ_VIRTUAL_HOST")
	//amqpConn, err := amqp.Dial("amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@" + os.Getenv("RABBITMQ_URL") + "/" + os.Getenv("RABBITMQ_VIRTUAL_HOST"))
	amqpConn, err := amqp.Dial(rabbitMQconnectString)

	failOnError(err, "Failed to connect to RabbitMQ")
	defer amqpConn.Close()

	ch, err := amqpConn.Channel()
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

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	//client := &http.Client{Timeout: time.Duration(3) * time.Second}

	go func() {
		for msg := range msgs {
			requestUrl := string(msg.Body[:])
			md5hash := getMD5Hash(requestUrl)

			log.Printf("Received a URL from RabbitMQ: %s\n", requestUrl)
			log.Printf("URL md5 hash: %s\n", md5hash)

			res, err := http.Get(requestUrl)
			if err != nil {
				fmt.Printf("error making http request: %s\n", err)
				// here ww need to make again request after 15 minutes
				os.Exit(1)
			}
			log.Printf("client: Response status code %d", res.StatusCode)

			if res.StatusCode != http.StatusOK {
				// if we have http.StatusOk in response we need to save data to DB table
				// if http.StatusError we need to send request again after 15 minutes

			} else {
				// working code for body
				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println("client: Response Body:")
				fmt.Println(string(body))

				//dumpBody, err := httputil.DumpResponse(res, true)
				//if err != nil {
				//	failOnError(err, "Failed to make response body dump.")
				//}

				fmt.Println("===========================")
				fmt.Println("client: Response headers:")
				builder := strings.Builder{}
				//var headersString string
				for h, v := range res.Header {
					for _, val := range v {
						builder.WriteString(fmt.Sprintf("%s: %s \n", h, val))
						//headersString += string(h) + ": " + val + "\n"
					}
				}
				hs := builder.String()
				fmt.Println(hs)
				fmt.Println("===========================")
				//res, err = client.Head(requestUrl)
				//if err != nil {
				//	log.Fatal(err)
				//}
				//for k, v := range res.Header {
				//	fmt.Printf("%s %s\n", k, v)
				//}

				// save data to DB urls table
				//respData := newResponseData(
				//	getMD5Hash(requestUrl),
				//	requestUrl,
				//	res.StatusCode,
				//	headersString,
				//	dumpBody,
				//)
				//fmt.Println("client: Dump respData:", respData)
			}

			// Remove message from RabbitMQ urls queue
			msg.Ack(false)

			log.Println("Done, waiting 30 sec.")
			log.Println("====================================================")
			time.Sleep(30 * time.Second)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

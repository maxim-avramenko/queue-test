package main

import (
	"context"
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

const sqlDriver = "mysql"
const amqpProtocol = "amqp"

type responseData struct {
	md5hash         string
	url             string
	response_status int
	headers         string
	body            string
}

func newResponseData(md5hash string, url string, responseStatus int, headers string, body string) *responseData {
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

func insert(db *sql.DB, rd *responseData) error {
	//query := "INSERT INTO product(product_name, product_price) VALUES (?, ?)"
	query := "INSERT INTO urls(md5hash, url, response_status, headers, body) VALUES (?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 1800*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	failOnError(err, "client: Error when preparing SQL statement")
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, rd.md5hash, rd.url, rd.response_status, rd.headers, rd.body)
	failOnError(err, "client: Error when inserting row into products table.")
	rows, err := res.RowsAffected()
	failOnError(err, "client: Error when finding rows affected.")
	log.Printf("client: %d products created ", rows)
	return err
}

// Необходимо добавить функцию переподключения к БД если она недоступна в данный момент
func getMySqlDSN() string {
	builder := strings.Builder{}
	builder.WriteString(os.Getenv("DBUSER"))
	builder.WriteString(":")
	builder.WriteString(os.Getenv("DBPASS"))
	builder.WriteString("@")
	builder.WriteString(os.Getenv("DBURL"))
	builder.WriteString("/")
	builder.WriteString(os.Getenv("DBNAME"))
	builder.WriteString("?")
	builder.WriteString("charset=utf8mb4,utf8")
	return builder.String()
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

	db, err := sql.Open(sqlDriver, getMySqlDSN())
	defer db.Close()
	failOnError(err, "Failed to connect to MariaDB")
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	amqpConn, err := amqp.Dial(getRabbitMqDSN())
	defer amqpConn.Close()
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := amqpConn.Channel()
	defer ch.Close()
	failOnError(err, "Failed to open a channel")

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
			failOnError(err, "Error making http request")
			//if err != nil {
			//	fmt.Printf("error making http request: %s\n", err)
			//	// here ww need to make again request after 15 minutes
			//	os.Exit(1)
			//}
			log.Printf("client: Response status code %d", res.StatusCode)

			if res.StatusCode != http.StatusOK {
				// if we have http.StatusOk in response we need to save data to DB table
				// if http.StatusError we need to send request again after 15 minutes
				log.Printf("client: Lets try again after 15 minutes, status code: %d", res.StatusCode)

			} else {
				// working code for body
				body, err := ioutil.ReadAll(res.Body)
				failOnError(err, "client: Failed to get response from remote server.")
				fmt.Println("client: Response Body:")
				bodyString := string(body)
				fmt.Println(bodyString)

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
				headersString := builder.String()
				fmt.Println(headersString)
				fmt.Println("===========================")
				//res, err = client.Head(requestUrl)
				//if err != nil {
				//	log.Fatal(err)
				//}
				//for k, v := range res.Header {
				//	fmt.Printf("%s %s\n", k, v)
				//}

				// save data to DB urls table
				respData := newResponseData(
					getMD5Hash(requestUrl),
					requestUrl,
					res.StatusCode,
					headersString,
					bodyString,
				)
				err = insert(db, respData)
				failOnError(err, "client: Error when get rows affected after insert. ")
			}

			// Remove message from RabbitMQ urls queue
			msg.Ack(false)

			log.Println("Done, waiting 30 sec.")
			log.Println("====================================================")
			time.Sleep(1 * time.Second)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

//db, err := sql.Open(
//	sqlDriver,
//	os.Getenv("DBUSER")+":"+os.Getenv("DBPASS")+"@"+os.Getenv("DBURL")+"/"+os.Getenv("DBNAME")+"?charset=utf8mb4,utf8")
//if err != nil {
//	panic(err)
//}

// See "Important settings" section.

// Capture rabbitmq connection properties.
//rabbitMQconnectString := "amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@" + os.Getenv("RABBITMQ_URL") + "/" + os.Getenv("RABBITMQ_VIRTUAL_HOST")
//amqpConn, err := amqp.Dial("amqp://" + os.Getenv("RABBITMQ_USER") + ":" + os.Getenv("RABBITMQ_PASSWORD") + "@" + os.Getenv("RABBITMQ_URL") + "/" + os.Getenv("RABBITMQ_VIRTUAL_HOST"))
//amqpConn, err := amqp.Dial(rabbitMQconnectString)

//fmt.Println("client: Dump respData:", respData)
//querySql := "INSERT INTO urls(md5hash, url, response_status, headers, body) VALUES ('?', '?', ?, '?', '?')"
//result, err := db.Query(querySql, md5hash, requestUrl, res.StatusCode, headersString, bodyString)
//failOnError(err, "Failed to insert data to DB")
//fmt.Printf("mysql insert result: ")
//fmt.Println(result)
//defer result.Close()
//result, err := db.Exec(querySql)
//if err != nil {
//	log.Fatal(err)
//}

//lastId, err := result.Scan()
//if err != nil {
//	log.Fatal(err)
//}
//
//fmt.Printf("The last inserted row id: %d\n", lastId)

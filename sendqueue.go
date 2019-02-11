package main

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// SendQueue struct...
type SendQueue struct {
	URL       string
	QueueName string
}

func failOnError(err error, msg string) {
	if err != nil {
		fmt.Printf("%s: %s", msg, err)
	}
}

// Close for
func (s SendQueue) Close() {
	//q.conn.Close()
	//q.ch.Close()
}

// Connect for
func (s SendQueue) Connect() *amqp.Channel {
	conn, err := amqp.Dial(s.URL)
	//defer conn.Close()
	if err != nil {
		failOnError(err, "Failed to connect to RabbitMQ")
		return nil
	}

	ch, err := conn.Channel()
	//defer ch.Close()
	if err != nil {
		failOnError(err, "Failed to open a channel")
		return nil
	}

	return ch
}

// Send for send message to queue
func (s SendQueue) Send(ch *amqp.Channel, msgID string, appID string, msgType string, msgBody string) bool {

	/*conn, err := amqp.Dial(q.URL)
	defer conn.Close()
	if err != nil {
		failOnError(err, "Failed to connect to RabbitMQ")
		return false
	}

	ch, err := conn.Channel()
	defer ch.Close()
	if err != nil {
		failOnError(err, "Failed to open a channel")
		return false
	}*/

	q, err := ch.QueueDeclare(
		s.QueueName, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		failOnError(err, "Failed to declare a queue")
		return false
	}

	body := msgBody
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			MessageId:    msgID,
			AppId:        appID,
			ContentType:  msgType,
			Body:         []byte(body),
		})
	log.Printf("## Sent a message : %s", body)
	//log.Printf("## >> Order Id :%s", msgID)
	if err != nil {
		failOnError(err, "Failed to publish a message")
		return false
	}

	return true
}

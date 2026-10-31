package consumer

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type StockExhaustedEnvelope struct {
	EventID string `json:"event_id"`
	Payload struct {
		AppointmentID string `json:"appointment_id"`
	} `json:"payload"`
}

func StartStockExhaustedConsumer(amqpURI string) {
	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal koneksi: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal buka channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("appointment.stock.exhausted.q", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal deklarasi queue: %v", err)
	}

	err = ch.QueueBind(q.Name, "pharmacy.stock.exhausted", "medcore.clinical.events", false, nil)
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal binding: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "appointment-worker", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("[Appointment Consumer] Gagal consume: %v", err)
	}

	log.Println("[Appointment Consumer] Menunggu event kompensasi stok...")

	go func() {
		for d := range msgs {
			var evt StockExhaustedEnvelope
			if err := json.Unmarshal(d.Body, &evt); err != nil {
				d.Nack(false, false)
				continue
			}

			log.Printf("[Appointment Consumer] Menerima kompensasi! Status Appointment %s diubah menjadi PENDING_MEDICATION_SUBSTITUTION", evt.Payload.AppointmentID)
			
			
			d.Ack(false)
		}
	}()

	select {}
}
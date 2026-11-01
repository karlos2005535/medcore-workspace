package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	// Import package publisher untuk mengirim event kompensasi
	"pharmacy-service/publisher"
)

type EventEnvelope struct {
	EventID string `json:"event_id"`
	Payload struct {
		AppointmentID string `json:"appointment_id"`
		Prescription  struct {
			PrescriptionID string `json:"prescription_id"`
			Items          []struct {
				DrugCode string `json:"drug_code"`
				Quantity int    `json:"quantity"`
			} `json:"items"`
		} `json:"prescription"`
	} `json:"payload"`
}

type ProcessedEventRecord struct {
	EventID     string    `bson:"event_id"`
	ProcessedAt time.Time `bson:"processed_at"`
	Status      string    `bson:"status"`
}

func StartPrescriptionConsumer(amqpURI string, mongoClient *mongo.Client) {
	// Inisialisasi Redis untuk Idempotency
	idem := NewRedisIdempotency("localhost:6379")

	conn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("[Pharmacy Consumer] Gagal koneksi broker: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[Pharmacy Consumer] Gagal membuka channel: %v", err)
	}
	defer ch.Close()

	// 1. Deklarasikan Queue durable dengan DLX
	args := amqp.Table{
		"x-dead-letter-exchange": "medcore.deadletter.exchange",
	}
	q, err := ch.QueueDeclare(
		"pharmacy.prescription.order.q",
		true,  // Durable
		false, // Auto-delete
		false, // Exclusive
		false, // No-wait
		args,  // Arguments
	)
	if err != nil {
		log.Fatalf("[Pharmacy Consumer] Gagal deklarasi queue: %v", err)
	}

	// 2. Ikat Queue ke Exchange
	err = ch.QueueBind(
		q.Name,
		"clinical.consultation.*",
		"medcore.clinical.events",
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[Pharmacy Consumer] Gagal binding queue: %v", err)
	}

	// 3. Set QoS = 1
	err = ch.Qos(10, 0, false)
	if err != nil {
		log.Fatalf("[Pharmacy Consumer] Gagal set QoS: %v", err)
	}

	// 4. Consume
	msgs, err := ch.Consume(
		q.Name,
		"pharmacy-worker",
		false, // autoAck = FALSE (Manual ACK)
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[Pharmacy Consumer] Gagal subscribe: %v", err)
	}

	log.Printf("[Pharmacy Consumer] Menunggu pesan resep pada antrean: %s", q.Name)
	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var evt EventEnvelope
			if err := json.Unmarshal(d.Body, &evt); err != nil {
				log.Printf("[Pharmacy Consumer] Format pesan tidak valid, NACK tanpa requeue: %v", err)
				d.Nack(false, false) // Langsung ke DLQ
				continue
			}

			log.Println("---------------------------------------------------------")
			log.Printf("[Pharmacy Consumer] Menerima Resep Baru!")
			log.Printf(" Event ID        : %s", evt.EventID)
			log.Printf(" Appointment ID  : %s", evt.Payload.AppointmentID)
			log.Printf(" Prescription ID : %s", evt.Payload.Prescription.PrescriptionID)

			// Validasi Poison Message (Kuantitas <= 0)
			isPoison := false
			for _, item := range evt.Payload.Prescription.Items {
				log.Printf(" -> Item: %s | Qty: %d", item.DrugCode, item.Quantity)
				if item.Quantity <= 0 {
					isPoison = true
				}
			}

			if isPoison {
				log.Printf("[Pharmacy Consumer] Terdeteksi Poison Message! Mengirim NACK ke DLQ.")
				d.Nack(false, false) // requeue = FALSE -> DLX
				log.Println("---------------------------------------------------------")
				continue
			}

			// -------------------------------------------------------------
			// SIMULASI STOK HABIS (LANGKAH 2.4)
			// Ubah menjadi true jika ingin menguji pengiriman event kompensasi
			// -------------------------------------------------------------
			stokHabis := true
			if stokHabis {
				log.Printf("[Pharmacy Consumer] Stok kosong! Mengirim event kompensasi ke Appointment...")
				// Memanggil fungsi dari package publisher
				publisher.PublishStockExhausted(amqpURI, evt.Payload.AppointmentID, "MED-AMX-500")
				d.Ack(false) // Tetap ACK karena pesan berhasil ditangani
				log.Println("---------------------------------------------------------")
				continue
			}
			// -------------------------------------------------------------

			// Siapkan Context untuk Redis dan MongoDB
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			// --- REDIS FAST-PATH ---
			status, acquired, err := idem.TryAcquire(ctx, evt.EventID)
			if err != nil {
				log.Printf("[Pharmacy Consumer] Redis error: %v, requeue", err)
				d.Nack(false, true)
				cancel()
				continue
			}
			if !acquired {
				if status == "SUCCESS" {
					log.Printf("[Deduplication Guard - REDIS] Event %s sudah diproses, ACK.", evt.EventID)
					d.Ack(false)
					cancel()
					log.Println("---------------------------------------------------------")
					continue
				}
				log.Printf("[Deduplication Guard - REDIS] Event %s sedang diproses, requeue.", evt.EventID)
				d.Nack(false, true)
				cancel()
				log.Println("---------------------------------------------------------")
				continue
			}

			// --- MONGODB FALLBACK ---
			eventColl := mongoClient.Database("medcore_pharmacy_db").Collection("processed_events")
			var existing ProcessedEventRecord
			err = eventColl.FindOne(ctx, bson.M{"event_id": evt.EventID}).Decode(&existing)
			if err == nil {
				log.Printf("[Deduplication Guard - MONGO] Event %s duplikat! ACK.", evt.EventID)
				_ = idem.MarkSuccess(ctx, evt.EventID)
				d.Ack(false)
				cancel()
				log.Println("---------------------------------------------------------")
				continue
			}

			// Simulasi Logika Bisnis (Pemotongan Stok)
			log.Printf("[Business Logic] Memproses stok resep untuk Event ID: %s", evt.EventID)

			// Catat Event ID ke MongoDB
			_, err = eventColl.InsertOne(ctx, ProcessedEventRecord{
				EventID:     evt.EventID,
				ProcessedAt: time.Now(),
				Status:      "SUCCESS",
			})
			
			if err != nil {
				log.Printf("[Error] Gagal mencatat deduplikasi: %v. Requeue pesan.", err)
				d.Nack(false, true) // Requeue ke antrean utama
				cancel()
				log.Println("---------------------------------------------------------")
				continue
			}

			// --- TRANSAKSI SUKSES ---
			_ = idem.MarkSuccess(ctx, evt.EventID)
			d.Ack(false)
			cancel()
			log.Printf("[Pharmacy Consumer] Transaksi selesai sempurna dan Event %s dicatat.", evt.EventID)
			log.Println("---------------------------------------------------------")
		}
	}()

	<-forever
}
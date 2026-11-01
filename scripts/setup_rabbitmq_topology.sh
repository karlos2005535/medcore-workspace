#!/bin/sh
# ============================================================
# Script: setup_rabbitmq_topology.sh
# Deskripsi: Mendeklarasikan Topologi AMQP untuk MedCore
# (Exchange, Queue, Binding, dan Dead Letter Exchange)
# ============================================================

CONTAINER="medcore-message-broker"
USER="admin_medcore"
PASS="SecuredPassRabbit2026!"
VHOST="/medcore_vhost"

echo "Memulai deklarasi topologi RabbitMQ pada Vhost $VHOST..."

# 1. Deklarasi Dead Letter Exchange (DLX) dan Dead Letter Queue (DLQ)
echo "-> Membuat DLX dan DLQ..."
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare exchange name=medcore.deadletter.exchange type=fanout durable=true
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare queue name=pharmacy.dlq durable=true
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare binding source=medcore.deadletter.exchange destination=pharmacy.dlq

# 2. Deklarasi Main Topic Exchange
echo "-> Membuat Topic Exchange Utama..."
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare exchange name=medcore.clinical.events type=topic durable=true

# 3. Deklarasi Queue Utama beserta argumen DLX (Khusus Farmasi)
echo "-> Membuat Antrean Utama..."
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare queue name=pharmacy.prescription.order.q durable=true arguments='{"x-dead-letter-exchange":"medcore.deadletter.exchange"}'
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare queue name=notification.patient.alert.q durable=true
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare queue name=appointment.stock.exhausted.q durable=true

# 4. Deklarasi Binding (Routing Key)
echo "-> Menghubungkan Antrean dengan Exchange..."
# Binding Farmasi (menerima semua event konsultasi)
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare binding source=medcore.clinical.events destination=pharmacy.prescription.order.q routing_key="clinical.consultation.*"

# Binding Notifikasi (menerima SEMUA event klinis)
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare binding source=medcore.clinical.events destination=notification.patient.alert.q routing_key="clinical.#"

# Binding Appointment (menerima event kompensasi stok habis dari Farmasi)
docker exec $CONTAINER rabbitmqadmin -u $USER -p $PASS -V $VHOST declare binding source=medcore.clinical.events destination=appointment.stock.exhausted.q routing_key="pharmacy.stock.exhausted"

echo "=================================================="
echo "Topologi RabbitMQ berhasil dideklarasikan!"
echo "=================================================="
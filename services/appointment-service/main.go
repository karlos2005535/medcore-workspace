package main

import (
	"context"
	"log"
	"time"

	pb "appointment-service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	log.Println("[Appointment Service] Menginisialisasi koneksi gRPC ke Pharmacy Service...")

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Tidak dapat membentuk koneksi ke Pharmacy Service: %v", err)
	}
	defer conn.Close()

	client := pb.NewPharmacyServiceClient(conn)

	requestValid := &pb.CheckDrugRequest{
		DrugCode:       "MED-AMX-500",
		QuantityNeeded: 20,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	log.Printf("[RPC Call] Mengirim permintaan validasi obat: %s", requestValid.DrugCode)
	resp, err := client.CheckDrugAvailability(ctx, requestValid)
	if err != nil {
		st, _ := status.FromError(err)
		log.Fatalf("RPC Gagal: Code=%s, Message=%s", st.Code(), st.Message())
	}

	log.Println("================ HASIL RESPON gRPC ================")
	log.Printf("Kode Obat      : %s", resp.GetDrugCode())
	log.Printf("Tersedia       : %t", resp.GetIsAvailable())
	log.Printf("Stok Aktual    : %d", resp.GetCurrentStock())
	log.Printf("Harga Satuan   : Rp %.2f", resp.GetUnitPrice())
	log.Printf("Catatan Sistem : %s", resp.GetMessage())
	log.Println("==================================================")
}

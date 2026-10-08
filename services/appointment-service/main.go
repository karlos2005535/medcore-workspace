package main

import (
	"context"
	"log"
	"time"

	pb "appointment-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const pharmacyAddr = "localhost:50051"

func checkDrug(client pb.PharmacyServiceClient, code string, qty int32, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("[RPC] CheckDrugAvailability code=%s qty=%d timeout=%s", code, qty, timeout)

	resp, err := client.CheckDrugAvailability(ctx, &pb.CheckDrugRequest{
		DrugCode:       code,
		QuantityNeeded: qty,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			log.Printf("[ERROR] non-gRPC error: %v", err)
			return
		}

		switch st.Code() {
		case codes.NotFound:
			log.Printf("[CLINICAL WARNING] Kode obat tidak ditemukan: %s", st.Message())
		case codes.ResourceExhausted:
			log.Printf("[CLINICAL WARNING] Permintaan melebihi kuota farmasi: %s", st.Message())
		case codes.DeadlineExceeded:
			log.Printf("[CLINICAL WARNING] Farmasi timeout, verifikasi manual diperlukan: %s", st.Message())
		case codes.Unavailable:
			log.Printf("[CLINICAL WARNING] Pharmacy Service tidak tersedia: %s", st.Message())
		default:
			log.Printf("[RPC ERROR] code=%s message=%s", st.Code(), st.Message())
		}
		return
	}

	log.Printf("[OK] code=%s available=%t stock=%d price=%.2f msg=%s",
		resp.GetDrugCode(),
		resp.GetIsAvailable(),
		resp.GetCurrentStock(),
		resp.GetUnitPrice(),
		resp.GetMessage(),
	)
}

func main() {
	log.Println("[Appointment Service] Menginisialisasi koneksi gRPC ke Pharmacy Service...")

	conn, err := grpc.Dial(pharmacyAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Tidak dapat membentuk koneksi ke Pharmacy Service: %v", err)
	}
	defer conn.Close()

	client := pb.NewPharmacyServiceClient(conn)

	// Skenario 1: normal
	checkDrug(client, "MED-AMX-500", 20, 2*time.Second)

	// Skenario 2: NotFound
	checkDrug(client, "MED-TIDAK-ADA", 10, 2*time.Second)

	// Skenario 3: ResourceExhausted
	checkDrug(client, "MED-AMX-500", 10000, 2*time.Second)
	// Skenario 4: DeadlineExceeded (MED-SLOW-500 memicu sleep 2s di server)
        checkDrug(client, "MED-SLOW-500", 10, 500*time.Millisecond)	
}
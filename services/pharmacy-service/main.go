package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	pb "pharmacy-service/pb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	mongoURI        = "mongodb://adm_pharmacy_svc:SecuredPassPharm2026!@localhost:27018/medcore_pharmacy_db?authSource=admin"
	grpcPort        = ":50051"
	restPort        = ":8081"
	quotaThreshold  = 5000
	bottleneckDelay = 2 * time.Second
)

type server struct {
	pb.UnimplementedPharmacyServiceServer
	mongoClient *mongo.Client
}

type DrugDoc struct {
	DrugCode   string  `bson:"drug_code" json:"drug_code"`
	Name       string  `bson:"name" json:"name"`
	UnitPrice  float64 `bson:"unit_price" json:"unit_price"`
	TotalStock int32   `bson:"total_stock" json:"total_stock"`
}

// ============================================================
// gRPC Handler: CheckDrugAvailability
// ============================================================
func (s *server) CheckDrugAvailability(ctx context.Context, req *pb.CheckDrugRequest) (*pb.CheckDrugResponse, error) {
	log.Printf("[gRPC Server] CheckDrugAvailability code=%s qty=%d", req.GetDrugCode(), req.GetQuantityNeeded())

	if req.GetDrugCode() == "MED-SLOW-500" {
		time.Sleep(bottleneckDelay)
	}

	if req.GetQuantityNeeded() > quotaThreshold {
		return nil, status.Errorf(codes.ResourceExhausted,
			"Permintaan %d melebihi batas kuota farmasi (%d)", req.GetQuantityNeeded(), quotaThreshold)
	}

	if req.GetDrugCode() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "drug_code tidak boleh kosong")
	}

	coll := s.mongoClient.Database("medcore_pharmacy_db").Collection("drugs")

	var drug DrugDoc
	err := coll.FindOne(ctx, bson.M{"drug_code": req.GetDrugCode()}).Decode(&drug)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Errorf(codes.NotFound,
				"Obat dengan kode %s tidak ditemukan dalam inventaris", req.GetDrugCode())
		}
		return nil, status.Errorf(codes.Internal, "Database error: %v", err)
	}

	if drug.TotalStock < req.GetQuantityNeeded() {
		return &pb.CheckDrugResponse{
			DrugCode:     drug.DrugCode,
			IsAvailable:  false,
			CurrentStock: drug.TotalStock,
			UnitPrice:    drug.UnitPrice,
			Message: fmt.Sprintf("Stok tidak mencukupi. Tersedia: %d, Dibutuhkan: %d",
				drug.TotalStock, req.GetQuantityNeeded()),
		}, nil
	}

	return &pb.CheckDrugResponse{
		DrugCode:     drug.DrugCode,
		IsAvailable:  true,
		CurrentStock: drug.TotalStock,
		UnitPrice:    drug.UnitPrice,
		Message:      "Stok obat mencukupi",
	}, nil
}

// ============================================================
// gRPC Handler: ReservePrescriptionStock
// ============================================================
func (s *server) ReservePrescriptionStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	log.Printf("[gRPC Server] ReservePrescriptionStock prescription_id=%s", req.GetPrescriptionId())
	return &pb.ReserveStockResponse{
		ReservationSuccess:   true,
		TransactionReference: fmt.Sprintf("TRX-%d", time.Now().Unix()),
		ErrorDetail:          "",
	}, nil
}

// ============================================================
// REST Baseline (HTTP/1.1 JSON)
// ============================================================
func startRESTServer(mongoClient *mongo.Client) {
	http.HandleFunc("/api/v1/drugs/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		drugCode := r.URL.Query().Get("drug_code")
		if drugCode == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "drug_code wajib diisi"})
			return
		}

		coll := mongoClient.Database("medcore_pharmacy_db").Collection("drugs")
		var drug DrugDoc
		err := coll.FindOne(r.Context(), bson.M{"drug_code": drugCode}).Decode(&drug)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Obat tidak ditemukan"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"drug_code":     drug.DrugCode,
			"name":          drug.Name,
			"is_available":  drug.TotalStock >= 10,
			"current_stock": drug.TotalStock,
			"unit_price":    drug.UnitPrice,
			"message":       "Stok obat mencukupi",
		})
	})

	log.Println("MedCore Pharmacy REST Baseline aktif pada port :8081")
	if err := http.ListenAndServe(restPort, nil); err != nil {
		log.Printf("REST server error: %v", err)
	}
}

// ============================================================
// Main
// ============================================================
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Gagal terhubung ke MongoDB: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping gagal: %v", err)
	}
	log.Println("Berhasil terhubung ke MongoDB medcore_pharmacy_db")

	// Jalankan REST server paralel
	go startRESTServer(client)

	// Jalankan gRPC server
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Gagal membuka port %s: %v", grpcPort, err)
	}

	s := grpc.NewServer()
	pb.RegisterPharmacyServiceServer(s, &server{mongoClient: client})

	log.Println("==================================================")
	log.Println("MedCore Pharmacy gRPC Service aktif pada port :50051")
	log.Println("MedCore Pharmacy REST Service aktif pada port :8081")
	log.Println("==================================================")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Gagal menjalankan gRPC server: %v", err)
	}
}
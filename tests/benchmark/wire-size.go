package main

import (
	"encoding/json"
	"fmt"

	pb "pharmacy-service/pb"
	"google.golang.org/protobuf/proto"
)

func main() {
	respProto := &pb.CheckDrugResponse{
		DrugCode:     "MED-AMX-500",
		IsAvailable:  true,
		CurrentStock: 500,
		UnitPrice:    3500.0,
		Message:      "Stok obat mencukupi",
	}
	protoBytes, _ := proto.Marshal(respProto)

	respJSON := map[string]interface{}{
		"drug_code":     "MED-AMX-500",
		"is_available":  true,
		"current_stock": 500,
		"unit_price":    3500.0,
		"message":       "Stok obat mencukupi",
	}
	jsonBytes, _ := json.Marshal(respJSON)

	fmt.Println("================ ANALISIS WIRE-SIZE PAYLOAD ================")
	fmt.Printf("Ukuran Payload JSON (HTTP/1.1)   : %d Bytes\n", len(jsonBytes))
	fmt.Printf("Ukuran Payload Protobuf (HTTP/2) : %d Bytes\n", len(protoBytes))
	efficiency := (1.0 - float64(len(protoBytes))/float64(len(jsonBytes))) * 100.0
	fmt.Printf("Efisiensi Reduksi Ukuran Kawat   : %.2f%%\n", efficiency)
	fmt.Println("============================================================")
}
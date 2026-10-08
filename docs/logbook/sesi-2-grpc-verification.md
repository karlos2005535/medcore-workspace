\# Logbook Sesi II — Verifikasi gRPC Sinkron Inter-Service



\*\*Modul Praktikum\*\*: 2 — Dekomposisi Domain, Isolasi Basis Data Terdistribusi, dan Komunikasi Sinkron Inter-Service (gRPC vs RESTful API)

\*\*Mata Kuliah\*\*: Arsitektur Mikroservis (MK41)

\*\*Pertemuan\*\*: 1, 2, dan 3 (Blok Fondasi Desain Terdistribusi \& Komunikasi Sinkron)

\*\*Tanggal\*\*: 8 Oktober 2026

\*\*Penanggung Jawab\*\*: Mahasiswa B — Appointment Service (Go)



\---



\## 1. Tujuan Sesi II



1\. Menyusun kontrak antarmuka `.proto` sebagai sumber kebenaran tunggal (single source of truth) komunikasi antar layanan.

2\. Mengompilasi stub/skeleton Protocol Buffers lintas bahasa (Go dan TypeScript).

3\. Mengimplementasikan gRPC Server pada Pharmacy Service dan gRPC Client pada Appointment Service.

4\. Memverifikasi komunikasi sinkron point-to-point dengan penanganan \*\*gRPC Standard Status Code\*\*.

5\. Membuktikan penanganan error terstandarisasi: `OK`, `NotFound`, `ResourceExhausted`, dan `DeadlineExceeded`.



\---



\## 2. Konteks Arsitektur



\### 2.1 Bounded Context Terkait



| Bounded Context | Service | Bahasa | Peran di Sesi II |

|---|---|---|---|

| Appointment Context | `appointment-service` | Go | \*\*gRPC Client\*\* — memverifikasi ketersediaan stok obat saat pembuatan janji temu |

| Pharmacy Context | `pharmacy-service` | Go + MongoDB | \*\*gRPC Server\*\* — memeriksa stok obat dari dokumen agregat |



Komunikasi antar kedua layanan bersifat \*\*sinkron point-to-point\*\* menggunakan \*\*gRPC di atas HTTP/2\*\* dengan serialisasi \*\*Protocol Buffers (binary)\*\*.



\### 2.2 Alasan Pemilihan gRPC



\- \*\*Kontrak eksplisit\*\* lewat file `.proto` — meminimalisir salah paham antar tim.

\- \*\*Code generation otomatis\*\* — stub client \& server dihasilkan dari satu file kontrak.

\- \*\*Serialisasi biner\*\* — payload lebih kecil daripada JSON teks.

\- \*\*HTTP/2 multiplexing\*\* — satu koneksi TCP dapat membawa banyak stream paralel.

\- \*\*Standard status code\*\* — error terstruktur dan dapat dipetakan ke aksi bisnis.



\---



\## 3. Kontrak Antarmuka — `proto/pharmacy.proto`



```proto

syntax = "proto3";



package medcore.pharmacy.v1;

option go\_package = "services/pharmacy-service/pb;pharmacypb";



service PharmacyService {

&#x20; rpc CheckDrugAvailability (CheckDrugRequest) returns (CheckDrugResponse);

&#x20; rpc ReservePrescriptionStock (ReserveStockRequest) returns (ReserveStockResponse);

}



message CheckDrugRequest {

&#x20; string drug\_code = 1;

&#x20; int32 quantity\_needed = 2;

}



message CheckDrugResponse {

&#x20; string drug\_code = 1;

&#x20; bool is\_available = 2;

&#x20; int32 current\_stock = 3;

&#x20; double unit\_price = 4;

&#x20; string message = 5;

}



message StockItem {

&#x20; string drug\_code = 1;

&#x20; int32 quantity = 2;

}



message ReserveStockRequest {

&#x20; string prescription\_id = 1;

&#x20; string appointment\_id = 2;

&#x20; repeated StockItem items = 3;

}



message ReserveStockResponse {

&#x20; bool reservation\_success = 1;

&#x20; string transaction\_reference = 2;

&#x20; string error\_detail = 3;

}


# Laporan Akhir Modul Praktikum 2

## Dekomposisi Domain, Isolasi Basis Data Terdistribusi, dan Komunikasi Sinkron Inter-Service (gRPC vs RESTful API)

**Mata Kuliah**: Arsitektur Mikroservis (MK41)
**Pertemuan**: 1, 2, dan 3 (Blok Fondasi Desain Terdistribusi & Komunikasi Sinkron)
**Beban Kredit**: 2 SKS Praktikum
**Tanggal**: 8 Oktober 2026
**Nama**: **\*\***Thomas Carlos Baco**\*\***
**NIM**: **\*\***42330041**\*\***
**Peran**: Mahasiswa B — Appointment Service (Go)
**Repository**: https://github.com/karlos2005535/medcore-workspace
**Branch**: `feature/appointment-service`

---

## Daftar Isi

1. [Pendahuluan](#1-pendahuluan)
2. [Landasan Teori](#2-landasan-teori)
3. [Arsitektur Sistem MedCore](#3-arsitektur-sistem-medcore)
4. [Sesi I — Dekomposisi Domain & Isolasi Multi-Instance DBMS](#4-sesi-i--dekomposisi-domain--isolasi-multi-instance-dbms)
5. [Sesi II — Kontrak Interface & Komunikasi gRPC Sinkron](#5-sesi-ii--kontrak-interface--komunikasi-grpc-sinkron)
6. [Sesi III — RESTful API Baseline & Benchmarking](#6-sesi-iii--restful-api-baseline--benchmarking)
7. [Analisis Non-Trivial](#7-analisis-non-trivial)
8. [Kesimpulan](#8-kesimpulan)
9. [Lampiran](#9-lampiran)

---

## 1. Pendahuluan

### 1.1 Latar Belakang

Sistem Informasi Manajemen Rumah Sakit (SIMRS) monolitik tradisional memiliki keterbatasan fundamental: seluruh modul (pendaftaran, rekam medis, farmasi, billing) berbagi satu basis data dengan relasi _foreign key_ global. Ketika volume transaksi meningkat, penguncian baris (_row-level lock_) pada satu tabel dapat melumpuhkan seluruh transaksi modul lain. Fenomena ini dikenal sebagai **Shared Database Anti-Pattern**.

Praktikum ini menerapkan **Strategic Domain-Driven Design (DDD)** untuk memecah sistem monolitik menjadi empat **Bounded Context** mandiri dengan _Database-per-Service_, lalu menghubungkannya lewat **komunikasi sinkron gRPC** di atas HTTP/2.

### 1.2 Tujuan Praktikum

1. Mendekomposisi entitas SIMRS monolitik menjadi 4 Bounded Context mandiri tanpa FK fisik antarlayanan.
2. Mengonfigurasi 4 instance DBMS terisolasi (PostgreSQL, MySQL, MongoDB) pada Docker virtual network.
3. Membuktikan fenomena Shared Database Anti-Pattern (lock contention).
4. Menyusun kontrak interface `.proto` dan mengompilasi stub lintas bahasa (Go dan TypeScript).
5. Mengimplementasikan pemanggilan RPC sinkron dengan penanganan status error terstandarisasi.
6. Menganalisis komparasi performa REST vs gRPC menggunakan k6.

### 1.3 Ruang Lingkup

Laporan ini fokus pada **peran Mahasiswa B — Appointment Service**, tetapi menyertakan konteks seluruh tim untuk kelengkapan analisis.

---

## 2. Landasan Teori

### 2.1 Strategic Domain-Driven Design (DDD)

DDD memisahkan sistem menjadi **Bounded Context**: batas eksplisit di mana sebuah model domain berlaku.

| Bounded Context         | Tanggung Jawab                             | Entitas Kunci         |
| ----------------------- | ------------------------------------------ | --------------------- |
| **Patient Context**     | Siklus hidup data demografi & kredensial   | Patient (full entity) |
| **Appointment Context** | Ketersediaan dokter & antrean poliklinik   | Appointment, Doctor   |
| **Pharmacy Context**    | Persediaan batch obat & pemenuhan resep    | Drug, Prescription    |
| **Billing Context**     | Faktur tagihan & verifikasi klaim asuransi | Invoice, Claim        |

### 2.2 Logical Identifier vs Physical Foreign Key

Dalam monolitik, relasi antar tabel memakai **foreign key fisik**. Dalam mikroservis dengan Database-per-Service, relasi ini diganti dengan **Logical Identifier**: UUID yang disimpan sebagai nilai biasa, tanpa constraint FK.

### 2.3 Pola Database-per-Service

| Aspek                    | Database-per-Service                       | Shared Database                       |
| ------------------------ | ------------------------------------------ | ------------------------------------- |
| **Otonomi Skema**        | Bebas migrasi tanpa koordinasi             | Perlu koordinasi ketat                |
| **Polyglot Persistence** | Bisa beda DBMS per service                 | Terbatas satu DBMS                    |
| **Fault Isolation**      | Gangguan satu DB tidak menjatuhkan lainnya | Gangguan satu tabel melumpuhkan semua |
| **Konsistensi**          | Eventual (butuh saga)                      | Immediate (ACID)                      |

### 2.4 Teorema CAP

MedCore memilih **CP** (Consistency + Partition Tolerance) karena sistem medis tidak boleh memberikan resep/tagihan berdasarkan data stale.

### 2.5 gRPC vs RESTful API

| Aspek           | gRPC                      | RESTful            |
| --------------- | ------------------------- | ------------------ |
| Protokol        | HTTP/2                    | HTTP/1.1           |
| Serialisasi     | Protocol Buffers (binary) | JSON (text)        |
| Kontrak         | `.proto` (IDL)            | OpenAPI (opsional) |
| Streaming       | Bidirectional             | Terbatas           |
| Browser Support | Perlu grpc-web            | Native             |

---

## 3. Arsitektur Sistem MedCore

### 3.1 Topologi Container

# Logbook Sesi I — Lock Contention & Break-The-Glass

**Modul Praktikum**: 2 — Dekomposisi Domain, Isolasi Basis Data Terdistribusi, dan Komunikasi Sinkron Inter-Service
**Mata Kuliah**: Arsitektur Mikroservis (MK41)
**Tanggal**: 8 Oktober 2026
**Nama**: Thomas Carlos Baco
**NIM**: 42330041
**Peran**: Mahasiswa B — Appointment Service (Go) & Mahasiswa C — Pharmacy Service (Go)
**Repository**: https://github.com/karlos2005535/medcore-workspace

---

## 1. Tujuan

1. Membuktikan fenomena **Shared Database Anti-Pattern** (table lock contention).
2. Memvalidasi **isolasi kredensial L7** antar Bounded Context.
3. Menguji protokol **Break-The-Glass** untuk pasien darurat tanpa identitas.

---

## 2. Verifikasi 4 Instance DBMS Terisolasi

**Perintah:**

```powershell
docker compose -f docker-compose.db.yml ps
```

**Output**
NAME IMAGE STATUS PORTS
medcore-appointment-db postgres:16-alpine Up 0.0.0.0:5433->5432/tcp
medcore-billing-db mysql:8.0 Up 0.0.0.0:3307->3306/tcp
medcore-patient-db postgres:16-alpine Up 0.0.0.0:5442->5432/tcp
medcore-pharmacy-db mongo:7.0-jammy Up 0.0.0.0:27018->27017/tcp

**Verifikasi DDL Appointment Context**
_perintah_
docker exec -it medcore-appointment-db psql -U usr*appointment_svc -d medcore_appointment_db -c "\dt"
docker exec -it medcore-appointment-db psql -U usr_appointment_svc -d medcore_appointment_db -c "SELECT * FROM doctors;"
docker exec -it medcore-appointment-db psql -U usr*appointment_svc -d medcore_appointment_db -c "SELECT * FROM appointments;"

**Output**
List of relations
Schema | Name | Type | Owner
--------+--------------+-------+---------------------
public | appointments | table | usr_appointment_svc
public | doctors | table | usr_appointment_svc

              doctor_id               | license_number |      doctor_name      | specialization

--------------------------------------+----------------+-----------------------+----------------
b0000000-0000-0000-0000-000000000001 | SIP-BALI-9901 | dr. Ketut Gede, Sp.PD | Penyakit Dalam
b0000000-0000-0000-0000-000000000002 | SIP-BALI-9902 | dr. Putu Ayu, Sp.A | Spesialis Anak

            appointment_id            |              patient_id              | consultation_status | clinical_notes

--------------------------------------+--------------------------------------+---------------------+----------------
b1111111-1111-1111-1111-111111111111 | a0000000-0000-0000-0000-000000000001 | SCHEDULED |

**Uji Isolasi Kredensial**

Perintah:
docker exec -it medcore-patient-db psql -h medcore-appointment-db -U usr_patient_svc -d medcore_appointment_db
ouput :
Password for user usr_patient_svc:
psql: error: connection to server at "medcore-appointment-db" (172.28.0.12), port 5432 failed:
FATAL: password authentication failed for user "usr_patient_svc"

**Simulasi Lock Contention**

**Terminal 1 : Tahan Lock**
docker exec -it medcore-appointment-db psql -U usr_appointment_svc -d medcore_appointment_db

**Di dalam psql**
BEGIN;
SELECT \* FROM appointments
WHERE appointment_id = 'b1111111-1111-1111-1111-111111111111'
FOR UPDATE;

**Terminal 1 — Lepaskan Lock**
ROLLBACK; >>" Terminal 2 langsung selesai"
**Output Terminal 2**
UPDATE 1

**Penjelasan**
Transaksi panjang di satu modul mengunci baris data yang dipakai modul lain. Ini bukti empiris bahaya Shared Database Anti-Pattern.

**Studi Kasus Break-The-Glass**
Skenario: Pasien darurat trauma masuk IGD tanpa identitas (Mr. X).

**Insert Appointment Darurat**
INSERT INTO appointments (
appointment_id, patient_id, doctor_id,
schedule_time, queue_number, consultation_status, clinical_notes
) VALUES (
'b2222222-2222-2222-2222-222222222222',
'00000000-9999-9999-9999-000000000000',
'b0000000-0000-0000-0000-000000000001',
CURRENT_TIMESTAMP, 999, 'IN_PROGRESS',
'EMERGENCY UNIDENTIFIED PATIENT'
); berarti : Insert berhasil

**Output**
INSERT 0 1

**Verifikasi Tidak Ada FK Error**
SELECT appointment_id, patient_id, consultation_status, clinical_notes
FROM appointments
WHERE appointment_id = 'b2222222-2222-2222-2222-222222222222';

**Output**
appointment_id | patient_id | consultation_status | clinical_notes
--------------------------------------+--------------------------------------+---------------------+-------------------------------
b2222222-2222-2222-2222-222222222222 | 00000000-9999-9999-9999-000000000000 | IN_PROGRESS | EMERGENCY UNIDENTIFIED PATIENT

**Rekonsiliasi Setelah Pasien Riil Terdaftar**
UPDATE appointments
SET patient_id = 'a0000000-0000-0000-0000-000000000003',
clinical_notes = COALESCE(clinical_notes,'') || E'\n[RECONCILED] updated from emergency temp ID at ' || NOW()
WHERE appointment_id = 'b2222222-2222-2222-2222-222222222222';

**output**
UPDATE 1

**Verifikasi Final**
SELECT appointment_id, patient_id, consultation_status, clinical_notes
FROM appointments
ORDER BY created_at;

**output**
appointment_id | patient_id | consultation_status | clinical_notes
--------------------------------------+--------------------------------------+---------------------+--------------------------------------------------------------------------------
b1111111-1111-1111-1111-111111111111 | a0000000-0000-0000-0000-000000000001 | SCHEDULED |
b2222222-2222-2222-2222-222222222222 | a0000000-0000-0000-0000-000000000003 | IN_PROGRESS | EMERGENCY UNIDENTIFIED PATIENT +
| | | [RECONCILED] updated from emergency temp ID at 2026-10-07 04:29:09.409822+00

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE patients (
    patient_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    national_id VARCHAR(16) UNIQUE NOT NULL, -- NIK 16 Digit
    medical_record_no VARCHAR(15) UNIQUE NOT NULL, -- Nomor Rekam Medis 

    full_name VARCHAR(150) NOT NULL,
    date_of_birth DATE NOT NULL,
    gender VARCHAR(10) CHECK (gender IN ('MALE', 'FEMALE')),
    blood_type VARCHAR(3)
    phone_number VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP 
);

INSERT INTO patients (patient_id, national_id, medical_record_no,
full_name, date_of_birth, gender, blood_type, phone_number) 
VALUES
('a0000000-0000-0000-0000-000000000001', '5171010101850001', 
'RM-2026-0001', 'I Wayan Sudarma', '1985-05-15', 'MALE', 'O', 
'+6281234567890'),
('a0000000-0000-0000-0000-000000000002', '5171010202900002', 
'RM-2026-0002', 'Ni Made Rai', '1990-11-20', 'FEMALE', 'A', 
'+6281987654321');
USE medcore_billing_db;
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS invoices (
    invoice_id VARCHAR(36) PRIMARY KEY,
    patient_id VARCHAR(36) NOT NULL,
    appointment_id VARCHAR(36) NOT NULL,
    total_amount DECIMAL(12,2) NOT NULL,
    payment_status VARCHAR(20) NOT NULL DEFAULT 'UNPAID',
    issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_appt (appointment_id),
    INDEX idx_patient (patient_id),
    CONSTRAINT chk_payment_status
        CHECK (payment_status IN ('UNPAID','PAID','CANCELLED','REFUNDED'))
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS invoice_items (
    item_id VARCHAR(36) PRIMARY KEY,
    invoice_id VARCHAR(36) NOT NULL,
    description VARCHAR(255) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    FOREIGN KEY (invoice_id) REFERENCES invoices(invoice_id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS insurance_claims (
    claim_id VARCHAR(36) PRIMARY KEY,
    invoice_id VARCHAR(36) NOT NULL,
    patient_id VARCHAR(36) NOT NULL,
    insurer_code VARCHAR(30) NOT NULL,
    claim_status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    claim_amount DECIMAL(12,2) NOT NULL,
    submitted_at TIMESTAMP NULL,
    verified_at TIMESTAMP NULL,
    FOREIGN KEY (invoice_id) REFERENCES invoices(invoice_id) ON DELETE CASCADE,
    INDEX idx_claim_invoice (invoice_id),
    INDEX idx_claim_patient (patient_id),
    CONSTRAINT chk_claim_status
        CHECK (claim_status IN ('DRAFT','SUBMITTED','VERIFIED','REJECTED','PAID'))
) ENGINE=InnoDB;

INSERT IGNORE INTO invoices (invoice_id, patient_id, appointment_id, total_amount, payment_status)
VALUES ('inv-0001',
        'a0000000-0000-0000-0000-000000000001',
        'b1111111-1111-1111-1111-111111111111',
        150000.00, 'UNPAID');

INSERT IGNORE INTO invoice_items (item_id, invoice_id, description, amount)
VALUES ('item-1', 'inv-0001', 'Jasa Konsultasi Spesialis Penyakit Dalam', 100000.00),
       ('item-2', 'inv-0001', 'Biaya Administrasi Rawat Jalan', 50000.00);

INSERT IGNORE INTO insurance_claims (claim_id, invoice_id, patient_id, insurer_code, claim_status, claim_amount)
VALUES ('claim-0001', 'inv-0001',
        'a0000000-0000-0000-0000-000000000001',
        'BPJS', 'DRAFT', 150000.00);

-- 계정 (고객/머천트/시스템계정)
CREATE TABLE accounts (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    external_id     VARCHAR(64) NOT NULL UNIQUE,
    account_type    ENUM('USER','MERCHANT','SYSTEM') NOT NULL,
    status          ENUM('ACTIVE','SUSPENDED','CLOSED') NOT NULL DEFAULT 'ACTIVE',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_type_status (account_type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 잔액 (가용/홀드 분리)
CREATE TABLE balances (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    account_id      BIGINT UNSIGNED NOT NULL,
    currency        VARCHAR(8) NOT NULL DEFAULT 'KRWS',
    available       DECIMAL(20,8) NOT NULL DEFAULT 0,
    held            DECIMAL(20,8) NOT NULL DEFAULT 0,
    version         INT UNSIGNED NOT NULL DEFAULT 0,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_account_currency (account_id, currency),
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    CONSTRAINT chk_available CHECK (available >= 0),
    CONSTRAINT chk_held CHECK (held >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 원장 (불변, double-entry)
CREATE TABLE ledger_entries (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    tx_id           VARCHAR(64) NOT NULL,
    account_id      BIGINT UNSIGNED NOT NULL,
    entry_type      ENUM('DEBIT','CREDIT') NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    balance_after   DECIMAL(20,8) NOT NULL,
    description     VARCHAR(256),
    idempotency_key VARCHAR(128),
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_tx_id (tx_id),
    INDEX idx_account_created (account_id, created_at),
    UNIQUE KEY uk_idem (idempotency_key),
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    CONSTRAINT chk_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 입금 (Deposit/Mint)
CREATE TABLE deposits (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    account_id      BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    status          ENUM('PENDING_MINT','MINTING','MINTED','FAILED') NOT NULL DEFAULT 'PENDING_MINT',
    tx_hash         VARCHAR(66),
    confirmations   INT UNSIGNED DEFAULT 0,
    retry_count     INT UNSIGNED DEFAULT 0,
    error_message   TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    FOREIGN KEY (account_id) REFERENCES accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 결제 (Payment)
CREATE TABLE payments (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    payer_id        BIGINT UNSIGNED NOT NULL,
    payee_id        BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    status          ENUM('AUTHORIZED','CAPTURED','VOIDED','EXPIRED','SETTLED') NOT NULL,
    authorized_at   TIMESTAMP NULL,
    captured_at     TIMESTAMP NULL,
    settled_at      TIMESTAMP NULL,
    expires_at      TIMESTAMP NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_payer_status (payer_id, status),
    FOREIGN KEY (payer_id) REFERENCES accounts(id),
    FOREIGN KEY (payee_id) REFERENCES accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 환매 (Redeem/Burn)
CREATE TABLE redeems (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    account_id      BIGINT UNSIGNED NOT NULL,
    amount          DECIMAL(20,8) NOT NULL,
    status          ENUM('PENDING_BURN','BURNING','BURNED','PAYOUT_PENDING','SETTLED','FAILED') NOT NULL DEFAULT 'PENDING_BURN',
    tx_hash         VARCHAR(66),
    confirmations   INT UNSIGNED DEFAULT 0,
    bank_ref        VARCHAR(64),
    retry_count     INT UNSIGNED DEFAULT 0,
    error_message   TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    FOREIGN KEY (account_id) REFERENCES accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Outbox (트랜잭셔널 아웃박스 패턴)
CREATE TABLE outbox_jobs (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    job_type        ENUM('MINT','BURN','PAYOUT') NOT NULL,
    reference_id    BIGINT UNSIGNED NOT NULL,
    payload         JSON NOT NULL,
    status          ENUM('PENDING','PROCESSING','COMPLETED','FAILED','DEAD_LETTER') NOT NULL DEFAULT 'PENDING',
    retry_count     INT UNSIGNED DEFAULT 0,
    max_retries     INT UNSIGNED DEFAULT 5,
    next_retry_at   TIMESTAMP NULL,
    locked_until    TIMESTAMP NULL,
    error_message   TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status_next (status, next_retry_at),
    INDEX idx_job_type_ref (job_type, reference_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 감사 로그
CREATE TABLE audit_logs (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    actor_id        BIGINT UNSIGNED,
    action          VARCHAR(64) NOT NULL,
    resource_type   VARCHAR(32) NOT NULL,
    resource_id     BIGINT UNSIGNED,
    old_value       JSON,
    new_value       JSON,
    ip_address      VARCHAR(45),
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_resource (resource_type, resource_id),
    INDEX idx_actor (actor_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 멱등성 키 저장
CREATE TABLE idempotency_keys (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    client_id       VARCHAR(64) NOT NULL,
    idem_key        VARCHAR(128) NOT NULL,
    response        JSON,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      TIMESTAMP NOT NULL,
    UNIQUE KEY uk_client_key (client_id, idem_key),
    INDEX idx_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 시스템 계정 초기 데이터 (준비금 계정)
INSERT INTO accounts (external_id, account_type, status) VALUES
    ('SYSTEM_RESERVE', 'SYSTEM', 'ACTIVE'),
    ('SYSTEM_FEE', 'SYSTEM', 'ACTIVE');

INSERT INTO balances (account_id, currency, available, held) VALUES
    (1, 'KRWS', 0, 0),
    (2, 'KRWS', 0, 0);

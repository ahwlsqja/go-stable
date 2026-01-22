# B2B Commerce Settlement Engine - 아키텍처 설계 문서

> 재고관리 + 주문처리 + 스테이블코인 정산 시스템

---

## 목차

1. [프로젝트 개요](#1-프로젝트-개요)
2. [기술 스택](#2-기술-스택)
3. [프로젝트 구조](#3-프로젝트-구조)
4. [DB 설계 (ERD)](#4-db-설계-erd)
5. [상태 머신 정의](#5-상태-머신-정의)
6. [정합성 보장 메커니즘](#6-정합성-보장-메커니즘)
7. [분산락 (Distributed Lock)](#7-분산락-distributed-lock)
8. [Double-Entry Ledger](#8-double-entry-ledger)
9. [Outbox 패턴](#9-outbox-패턴)
10. [트랜잭션 경계](#10-트랜잭션-경계)
11. [API 설계](#11-api-설계)
12. [에러 처리](#12-에러-처리)

---

## 1. 프로젝트 개요

### 1.1 미션

B2B 커머스 환경에서 **재고관리**, **주문처리**, **스테이블코인 기반 결제/정산**을 금융급 정합성으로 구현하는 것.

### 1.2 핵심 요구사항

| 구분 | 요구사항 |
|------|----------|
| **정합성** | 모든 거래는 원자적이며, 중간 상태가 외부에 노출되지 않음 |
| **멱등성** | 동일 요청의 재시도가 부작용 없이 동일 결과 반환 |
| **추적성** | 모든 상태 변경은 감사 로그로 기록 |
| **복원력** | 외부 시스템 장애 시 재시도 및 보상 트랜잭션 지원 |

### 1.3 핵심 도메인

```
┌─────────────────────────────────────────────────────────────────┐
│                        B2B Commerce Engine                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│   │   Product   │    │  Inventory  │    │    Order    │        │
│   │   (상품)    │───►│   (재고)    │◄───│   (주문)    │        │
│   └─────────────┘    └─────────────┘    └──────┬──────┘        │
│                                                 │                │
│                                                 ▼                │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│   │   Account   │◄───│   Payment   │───►│ Settlement  │        │
│   │   (계정)    │    │   (결제)    │    │   (정산)    │        │
│   └──────┬──────┘    └─────────────┘    └─────────────┘        │
│          │                                                       │
│          ▼                                                       │
│   ┌─────────────┐                                               │
│   │   Ledger    │  ← Double-Entry (차변 = 대변)                 │
│   │   (원장)    │                                               │
│   └─────────────┘                                               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 기술 스택

| 레이어 | 기술 | 선택 이유 |
|--------|------|----------|
| **Language** | Go 1.22+ | 동시성, 성능, 타입 안전성 |
| **HTTP** | Gin | 성숙한 생태계, 표준 net/http 호환 |
| **DB** | MySQL 8.0 + sqlc | 트랜잭션 지원, 컴파일 타임 쿼리 검증 |
| **Cache/Lock** | Redis 7 | 분산락, 캐시, 세션 관리 |
| **Chain** | Anvil + go-ethereum | 로컬 EVM 테스트, 스테이블코인 연동 |
| **Docs** | Swagger (swaggo) | API 문서 자동 생성 |

### 2.1 sqlc 선택 이유 (vs GORM)

| 항목 | sqlc | GORM |
|------|------|------|
| 쿼리 검증 | 컴파일 타임 | 런타임 |
| 쿼리 제어 | SQL 직접 작성 | ORM 추상화 |
| N+1 방지 | 명시적 JOIN | 주의 필요 |
| 성능 | Raw SQL 수준 | 오버헤드 존재 |
| 감사 적합성 | SQL 그대로 로깅 | 변환된 쿼리 |

**결론**: 금융 시스템에서는 **쿼리 투명성**과 **성능 예측 가능성**이 중요하므로 sqlc 선택

---

## 3. 프로젝트 구조

```
b2b-settlement-engine/
├── cmd/
│   ├── api/main.go              # HTTP API 서버 진입점
│   └── worker/main.go           # Outbox Worker 진입점
│
├── internal/
│   ├── product/                 # 상품 도메인
│   │   ├── handler.go           #   HTTP 핸들러
│   │   ├── service.go           #   비즈니스 로직
│   │   ├── repository.go        #   DB 접근
│   │   └── model.go             #   도메인 모델
│   │
│   ├── inventory/               # 재고 도메인
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── state.go             #   상태 머신
│   │
│   ├── order/                   # 주문 도메인
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── state.go             #   상태 머신
│   │
│   ├── payment/                 # 결제 도메인
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── state.go             #   상태 머신
│   │
│   ├── ledger/                  # 원장 도메인
│   │   ├── service.go           #   Double-entry 로직
│   │   └── repository.go
│   │
│   ├── settlement/              # 정산 도메인
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   │
│   ├── outbox/                  # Outbox Worker
│   │   ├── worker.go
│   │   ├── processor.go
│   │   └── repository.go
│   │
│   └── common/
│       ├── handler/             #   공통 핸들러 (health)
│       ├── middleware/          #   미들웨어
│       │   ├── request_id.go    #     요청 ID 생성
│       │   ├── logger.go        #     구조화 로깅
│       │   ├── idempotency.go   #     멱등성 처리
│       │   └── response.go      #     응답 헬퍼
│       └── errors/              #   표준 에러 타입
│
├── pkg/
│   ├── db/mysql.go              # MySQL 연결 관리
│   ├── redis/client.go          # Redis 클라이언트
│   ├── lock/distributed.go      # 분산락 구현
│   └── logger/zap.go            # 구조화 로거
│
├── db/
│   ├── migrations/              # DB 마이그레이션
│   │   ├── 000001_init.up.sql
│   │   └── 000001_init.down.sql
│   └── queries/                 # sqlc 쿼리
│       ├── product.sql
│       ├── inventory.sql
│       ├── order.sql
│       ├── payment.sql
│       └── ledger.sql
│
├── docs/                        # Swagger 생성 문서
├── contracts/                   # Solidity 컨트랙트
├── docker-compose.yml
├── Makefile
└── sqlc.yaml
```

### 3.1 레이어 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Request                            │
└─────────────────────────────┬───────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Middleware Layer                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ RequestID   │  │   Logger    │  │ Idempotency │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────┬───────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Handler Layer                           │
│  - HTTP 요청 파싱, 검증                                      │
│  - 응답 직렬화                                               │
│  - Swagger 문서화                                            │
└─────────────────────────────┬───────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  - 비즈니스 로직                                             │
│  - 트랜잭션 경계                                             │
│  - 상태 머신 전이                                            │
│  - 분산락 획득/해제                                          │
└─────────────────────────────┬───────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                          │
│  - DB CRUD                                                   │
│  - sqlc 생성 코드 래핑                                       │
└─────────────────────────────┬───────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Infrastructure                           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │  MySQL  │  │  Redis  │  │  Chain  │  │ Outbox  │        │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘        │
└─────────────────────────────────────────────────────────────┘
```

---

## 4. DB 설계 (ERD)

### 4.1 전체 ERD

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              B2B Commerce Settlement Engine ERD                       │
└─────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────┐       ┌──────────────────┐       ┌──────────────────┐
│   products   │       │   inventories    │       │  inventory_logs  │
├──────────────┤       ├──────────────────┤       ├──────────────────┤
│ PK id        │◄──────│ FK product_id    │◄──────│ FK inventory_id  │
│ UK sku       │       │ PK id            │       │ PK id            │
│    name      │       │ UK (product_id,  │       │    event_type    │
│    price     │       │    location)     │       │    quantity_change│
│    status    │       │    quantity      │       │    quantity_after │
│    created_at│       │    reserved_qty  │       │    reserved_after │
│    updated_at│       │    version (OL)  │       │    reference_type │
└──────────────┘       │    created_at    │       │    reference_id   │
        │              │    updated_at    │       │    reason         │
        │              └──────────────────┘       │    created_at     │
        │                                         └──────────────────┘
        │                                                 ▲
        │                                                 │ (reference)
        ▼                                                 │
┌──────────────────┐       ┌──────────────────┐          │
│   order_items    │       │     orders       │          │
├──────────────────┤       ├──────────────────┤          │
│ PK id            │       │ PK id            │          │
│ FK order_id      │──────►│ UK order_number  │          │
│ FK product_id    │       │    buyer_id      │          │
│    quantity      │       │    seller_id     │          │
│    unit_price    │       │    status (SM)   │──────────┘
│    created_at    │       │    total_amount  │
└──────────────────┘       │    created_at    │
                           │    updated_at    │
                           └────────┬─────────┘
                                    │
                                    ▼
┌──────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│    accounts      │       │    payments      │       │   settlements    │
├──────────────────┤       ├──────────────────┤       ├──────────────────┤
│ PK id            │◄──────│ FK payer_acct_id │       │ PK id            │
│ UK external_id   │       │ FK order_id      │◄──────│ FK payment_id    │
│    account_type  │       │ PK id            │       │ FK payee_acct_id │──┐
│    owner_id      │       │ UK idempotency_  │       │    amount        │  │
│    balance       │       │    key           │       │    fee_amount    │  │
│    hold_balance  │       │    amount        │       │    net_amount    │  │
│    version (OL)  │       │    status (SM)   │       │    status (SM)   │  │
│    status        │       │    authorized_at │       │    settled_at    │  │
│    created_at    │       │    captured_at   │       │    created_at    │  │
│    updated_at    │       │    expires_at    │       │    updated_at    │  │
└────────┬─────────┘       │    created_at    │       └──────────────────┘  │
         │                 │    updated_at    │                             │
         │                 └──────────────────┘                             │
         │                         │                                        │
         │                         │ (reference)                            │
         ▼                         ▼                                        │
┌──────────────────┐       ┌──────────────────┐                             │
│  ledger_entries  │       │     outbox       │                             │
├──────────────────┤       ├──────────────────┤                             │
│ PK id            │       │ PK id            │◄────────────────────────────┘
│ IDX tx_id        │       │    event_type    │
│ FK account_id    │       │    aggregate_type│
│    entry_type    │       │    aggregate_id  │
│    amount        │       │    payload (JSON)│
│    balance_after │       │    status        │
│    reference_type│       │    retry_count   │
│    reference_id  │       │    max_retries   │
│    description   │       │    next_retry_at │
│    created_at    │       │    error_message │
└──────────────────┘       │    created_at    │
                           │    updated_at    │
                           └──────────────────┘

┌──────────────────┐       ┌──────────────────┐
│idempotency_keys  │       │   audit_logs     │
├──────────────────┤       ├──────────────────┤
│ PK id            │       │ PK id            │
│ UK idempotency_  │       │    actor_type    │
│    key           │       │    actor_id      │
│    request_path  │       │    action        │
│    request_hash  │       │    resource_type │
│    response_stat │       │    resource_id   │
│    response_body │       │    old_value     │
│    created_at    │       │    new_value     │
│    expires_at    │       │    ip_address    │
└──────────────────┘       │    user_agent    │
                           │    request_id    │
                           │    created_at    │
                           └──────────────────┘

범례:
  PK  = Primary Key
  FK  = Foreign Key
  UK  = Unique Key
  OL  = Optimistic Lock (version 컬럼)
  SM  = State Machine (status ENUM)
  IDX = Index
```

### 4.2 테이블 상세 명세

#### 4.2.1 products (상품)

| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 상품 ID |
| sku | VARCHAR(50) | UNIQUE, NOT NULL | Stock Keeping Unit |
| name | VARCHAR(200) | NOT NULL | 상품명 |
| price | DECIMAL(18,2) | NOT NULL | 단가 |
| status | ENUM | NOT NULL, DEFAULT 'ACTIVE' | ACTIVE, INACTIVE |
| created_at | TIMESTAMP | NOT NULL | 생성 시각 |
| updated_at | TIMESTAMP | NOT NULL | 수정 시각 |

**인덱스:**
- `PRIMARY KEY (id)`
- `UNIQUE KEY (sku)`
- `INDEX idx_status (status)`

#### 4.2.2 inventories (재고)

| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 재고 ID |
| product_id | BIGINT UNSIGNED | FK, NOT NULL | 상품 참조 |
| location | VARCHAR(50) | NOT NULL, DEFAULT 'default' | 창고 위치 |
| quantity | BIGINT | NOT NULL, DEFAULT 0 | 총 수량 |
| reserved_quantity | BIGINT | NOT NULL, DEFAULT 0 | 예약 수량 |
| version | INT UNSIGNED | NOT NULL, DEFAULT 1 | Optimistic Lock |
| created_at | TIMESTAMP | NOT NULL | 생성 시각 |
| updated_at | TIMESTAMP | NOT NULL | 수정 시각 |

**인덱스:**
- `PRIMARY KEY (id)`
- `UNIQUE KEY uk_product_location (product_id, location)`

**CHECK 제약:**
```sql
CONSTRAINT chk_quantity CHECK (quantity >= 0)
CONSTRAINT chk_reserved CHECK (reserved_quantity >= 0)
CONSTRAINT chk_available CHECK (quantity >= reserved_quantity)
```

#### 4.2.3 inventory_logs (재고 이력)

| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 로그 ID |
| inventory_id | BIGINT UNSIGNED | FK, NOT NULL | 재고 참조 |
| event_type | ENUM | NOT NULL | INBOUND, OUTBOUND, RESERVE, RELEASE, ADJUST |
| quantity_change | BIGINT | NOT NULL | 변동량 (+/-) |
| quantity_after | BIGINT | NOT NULL | 변동 후 총 수량 |
| reserved_after | BIGINT | NOT NULL | 변동 후 예약 수량 |
| reference_type | VARCHAR(20) | | 참조 타입 (ORDER, ADJUSTMENT 등) |
| reference_id | BIGINT UNSIGNED | | 참조 ID |
| reason | VARCHAR(255) | | 사유 |
| created_at | TIMESTAMP | NOT NULL | 생성 시각 |

**특징:** INSERT ONLY (불변 테이블)

**인덱스:**
- `PRIMARY KEY (id)`
- `INDEX idx_inventory_created (inventory_id, created_at)`
- `INDEX idx_reference (reference_type, reference_id)`

#### 4.2.4 orders (주문)

| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 주문 ID |
| order_number | VARCHAR(50) | UNIQUE, NOT NULL | 주문 번호 |
| buyer_id | BIGINT UNSIGNED | NOT NULL | 구매자 ID |
| seller_id | BIGINT UNSIGNED | NOT NULL | 판매자 ID |
| status | ENUM | NOT NULL | 주문 상태 |
| total_amount | DECIMAL(18,2) | NOT NULL | 총 금액 |
| created_at | TIMESTAMP | NOT NULL | 생성 시각 |
| updated_at | TIMESTAMP | NOT NULL | 수정 시각 |

**status ENUM 값:**
- PENDING, CONFIRMED, PAID, SHIPPED, COMPLETED, CANCELLED, REFUNDED

**인덱스:**
- `PRIMARY KEY (id)`
- `UNIQUE KEY (order_number)`
- `INDEX idx_buyer_status (buyer_id, status)`
- `INDEX idx_seller_status (seller_id, status)`

#### 4.2.5 accounts (계정)

| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 계정 ID |
| account_type | ENUM | NOT NULL | USER, MERCHANT, ESCROW, SYSTEM |
| owner_id | BIGINT UNSIGNED | | 소유자 ID |
| external_id | VARCHAR(64) | UNIQUE | 외부 식별자 |
| balance | DECIMAL(18,8) | NOT NULL, DEFAULT 0 | 가용 잔액 |
| hold_balance | DECIMAL(18,8) | NOT NULL, DEFAULT 0 | 홀드 잔액 |
| version | INT UNSIGNED | NOT NULL, DEFAULT 1 | Optimistic Lock |
| status | ENUM | NOT NULL | ACTIVE, SUSPENDED, CLOSED |
| created_at | TIMESTAMP | NOT NULL | 생성 시각 |
| updated_at | TIMESTAMP | NOT NULL | 수정 시각 |

**CHECK 제약:**
```sql
CONSTRAINT chk_balance CHECK (balance >= 0)
CONSTRAINT chk_hold CHECK (hold_balance >= 0)
```

#### 4.2.6 ledger_entries (원장)

| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 원장 ID |
| tx_id | VARCHAR(64) | NOT NULL | 트랜잭션 ID (논리적) |
| account_id | BIGINT UNSIGNED | FK, NOT NULL | 계정 참조 |
| entry_type | ENUM | NOT NULL | DEBIT, CREDIT |
| amount | DECIMAL(18,8) | NOT NULL | 금액 (항상 양수) |
| balance_after | DECIMAL(18,8) | NOT NULL | 기록 시점 잔액 |
| reference_type | VARCHAR(30) | | 참조 타입 |
| reference_id | BIGINT UNSIGNED | | 참조 ID |
| description | VARCHAR(255) | | 설명 |
| created_at | TIMESTAMP | NOT NULL | 생성 시각 |

**특징:** INSERT ONLY (불변 테이블)

**CHECK 제약:**
```sql
CONSTRAINT chk_amount CHECK (amount > 0)
```

**인덱스:**
- `PRIMARY KEY (id)`
- `INDEX idx_tx_id (tx_id)` - tx_id별 차대변 검증
- `INDEX idx_account_created (account_id, created_at)` - 계정별 이력 조회

#### 4.2.7 payments (결제)

| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 결제 ID |
| idempotency_key | VARCHAR(128) | UNIQUE, NOT NULL | 멱등성 키 |
| order_id | BIGINT UNSIGNED | FK, NOT NULL | 주문 참조 |
| payer_account_id | BIGINT UNSIGNED | FK, NOT NULL | 지불자 계정 |
| amount | DECIMAL(18,8) | NOT NULL | 결제 금액 |
| status | ENUM | NOT NULL | 결제 상태 |
| authorized_at | TIMESTAMP | NULL | 승인 시각 |
| captured_at | TIMESTAMP | NULL | 캡처 시각 |
| expires_at | TIMESTAMP | NULL | 승인 만료 시각 |
| created_at | TIMESTAMP | NOT NULL | 생성 시각 |
| updated_at | TIMESTAMP | NOT NULL | 수정 시각 |

**status ENUM 값:**
- PENDING, AUTHORIZED, CAPTURED, VOIDED, REFUNDED, FAILED

#### 4.2.8 outbox (이벤트 아웃박스)

| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 이벤트 ID |
| event_type | VARCHAR(50) | NOT NULL | 이벤트 타입 |
| aggregate_type | VARCHAR(50) | NOT NULL | 집합 타입 |
| aggregate_id | BIGINT UNSIGNED | NOT NULL | 집합 ID |
| payload | JSON | NOT NULL | 이벤트 페이로드 |
| status | ENUM | NOT NULL | PENDING, PROCESSING, COMPLETED, FAILED, DEAD_LETTER |
| retry_count | INT UNSIGNED | NOT NULL, DEFAULT 0 | 재시도 횟수 |
| max_retries | INT UNSIGNED | NOT NULL, DEFAULT 5 | 최대 재시도 |
| next_retry_at | TIMESTAMP | NULL | 다음 재시도 시각 |
| error_message | TEXT | | 에러 메시지 |
| created_at | TIMESTAMP | NOT NULL | 생성 시각 |
| updated_at | TIMESTAMP | NOT NULL | 수정 시각 |

**인덱스:**
- `INDEX idx_status_retry (status, next_retry_at)` - Worker 폴링용

### 4.3 인덱스 전략

```sql
-- 1. 상품 조회 (상태별)
INDEX idx_status (status) ON products;

-- 2. 재고 조회 (상품+위치 유니크)
UNIQUE KEY uk_product_location (product_id, location) ON inventories;

-- 3. 재고 이력 조회 (시간순 페이징)
INDEX idx_inventory_created (inventory_id, created_at) ON inventory_logs;

-- 4. 주문 목록 (구매자/판매자별 + 상태)
INDEX idx_buyer_status (buyer_id, status) ON orders;
INDEX idx_seller_status (seller_id, status) ON orders;

-- 5. 원장 조회 (계정별 시간순)
INDEX idx_account_created (account_id, created_at) ON ledger_entries;

-- 6. tx_id별 차대변 검증
INDEX idx_tx_id (tx_id) ON ledger_entries;

-- 7. Outbox Worker 폴링
INDEX idx_status_retry (status, next_retry_at) ON outbox;

-- 8. 멱등성 키 만료 정리
INDEX idx_expires (expires_at) ON idempotency_keys;
```

---

## 5. 상태 머신 정의

### 5.1 재고 이벤트 (Inventory Event)

```
┌─────────────────────────────────────────────────────────────┐
│                    Inventory Events                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  INBOUND   : 입고 (quantity ↑)                              │
│  OUTBOUND  : 출고 (quantity ↓)                              │
│  RESERVE   : 예약 (reserved_quantity ↑)                     │
│  RELEASE   : 예약 해제 (reserved_quantity ↓)                │
│  ADJUST    : 재고 조정 (quantity ↑↓)                        │
│                                                              │
│  available_quantity = quantity - reserved_quantity          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 주문 상태 (Order Status)

```
                    ┌──────────────┐
                    │   PENDING    │ 주문 생성
                    └──────┬───────┘
                           │ confirm()
                           │ → 재고 RESERVE
                    ┌──────▼───────┐
              ┌─────│  CONFIRMED   │ 주문 확정
              │     └──────┬───────┘
              │            │ pay()
              │            │ → Payment AUTHORIZE
              │     ┌──────▼───────┐
              │     │     PAID     │ 결제 완료
              │     └──────┬───────┘
              │            │ ship()
              │            │ → 재고 OUTBOUND (RESERVE → 실제 차감)
              │     ┌──────▼───────┐
              │     │   SHIPPED    │ 배송 시작
              │     └──────┬───────┘
              │            │ complete()
              │            │ → Settlement 생성
              │     ┌──────▼───────┐
              │     │  COMPLETED   │ 주문 완료
              │     └──────────────┘
              │
   cancel()   │     ┌──────────────┐
   → 재고     └────►│  CANCELLED   │ 주문 취소
     RELEASE        └──────────────┘
                           ▲
                           │ refund() (from PAID)
                           │ → Payment REFUND
                    ┌──────┴───────┐
                    │   REFUNDED   │ 환불 완료
                    └──────────────┘
```

**상태 전이 매트릭스:**

| From \ To | CONFIRMED | PAID | SHIPPED | COMPLETED | CANCELLED | REFUNDED |
|-----------|:---------:|:----:|:-------:|:---------:|:---------:|:--------:|
| PENDING   | ✅        | ❌   | ❌      | ❌        | ✅        | ❌       |
| CONFIRMED | ❌        | ✅   | ❌      | ❌        | ✅        | ❌       |
| PAID      | ❌        | ❌   | ✅      | ❌        | ❌        | ✅       |
| SHIPPED   | ❌        | ❌   | ❌      | ✅        | ❌        | ❌       |
| COMPLETED | ❌        | ❌   | ❌      | ❌        | ❌        | ❌       |
| CANCELLED | ❌        | ❌   | ❌      | ❌        | ❌        | ❌       |
| REFUNDED  | ❌        | ❌   | ❌      | ❌        | ❌        | ❌       |

### 5.3 결제 상태 (Payment Status)

```
┌─────────┐  authorize()  ┌────────────┐  capture()   ┌──────────┐
│ PENDING │──────────────►│ AUTHORIZED │─────────────►│ CAPTURED │
└─────────┘               └─────┬──────┘              └────┬─────┘
                                │                          │
                          void()│                    refund()│
                                ▼                          ▼
                          ┌──────────┐              ┌──────────┐
                          │  VOIDED  │              │ REFUNDED │
                          └──────────┘              └──────────┘
```

**상태별 의미:**

| 상태 | 의미 | balance 영향 | hold_balance 영향 |
|------|------|-------------|------------------|
| PENDING | 결제 요청 생성 | - | - |
| AUTHORIZED | 결제 승인 (홀드) | - | +amount |
| CAPTURED | 결제 확정 | -amount | -amount (홀드 해제) |
| VOIDED | 승인 취소 | - | -amount (홀드 해제) |
| REFUNDED | 환불 완료 | +amount | - |

### 5.4 Outbox 상태 (Outbox Status)

```
┌─────────┐    poll()    ┌────────────┐   success()  ┌───────────┐
│ PENDING │─────────────►│ PROCESSING │─────────────►│ COMPLETED │
└────┬────┘              └─────┬──────┘              └───────────┘
     │                         │
     │                    fail()│
     │                         ▼
     │                   ┌──────────┐
     │                   │  FAILED  │ retry_count < max_retries
     │                   └────┬─────┘
     │                        │
     │    retry_count++       │ retry_count >= max_retries
     │    next_retry_at 설정  │
     │         ▲              ▼
     │         │        ┌─────────────┐
     └─────────┴───────►│ DEAD_LETTER │ 수동 처리 필요
                        └─────────────┘
```

---

## 6. 정합성 보장 메커니즘

### 6.1 Optimistic Lock (낙관적 잠금)

**적용 대상:** `inventories`, `accounts`

**원리:**
```sql
-- 1. 조회 시 version 함께 읽기
SELECT id, quantity, version FROM inventories WHERE id = 1;
-- 결과: quantity=100, version=1

-- 2. 업데이트 시 version 조건 추가
UPDATE inventories
SET quantity = quantity - 10,
    version = version + 1
WHERE id = 1
  AND version = 1;  -- 조회 시점의 version

-- 3. affected rows 확인
-- rows = 1: 성공
-- rows = 0: 다른 트랜잭션이 먼저 수정함 → 재시도
```

**동시성 시나리오:**

```
┌─────────────────────────────────────────────────────────────────┐
│                    Optimistic Lock Race Condition                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  시간   Transaction A              Transaction B                 │
│  ────   ─────────────              ─────────────                 │
│  T1     SELECT (qty=100, v=1)                                    │
│  T2                                 SELECT (qty=100, v=1)        │
│  T3     UPDATE qty=90, v=2                                       │
│         WHERE v=1                                                │
│         → rows=1 ✅                                              │
│  T4                                 UPDATE qty=80, v=2           │
│                                     WHERE v=1                    │
│                                     → rows=0 ❌                  │
│                                     → RETRY 또는 ERROR           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Go 구현 예시:**

```go
// repository/inventory.go
func (r *Repository) DeductQuantity(ctx context.Context, id int64, qty int64, version int) (int, error) {
    result, err := r.db.ExecContext(ctx, `
        UPDATE inventories
        SET quantity = quantity - ?,
            version = version + 1,
            updated_at = NOW()
        WHERE id = ?
          AND version = ?
          AND quantity >= ?
    `, qty, id, version, qty)

    if err != nil {
        return 0, err
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return 0, ErrOptimisticLock // 재시도 필요
    }

    return version + 1, nil
}

// service/inventory.go
func (s *Service) Deduct(ctx context.Context, id int64, qty int64) error {
    const maxRetries = 3

    for i := 0; i < maxRetries; i++ {
        inv, err := s.repo.Get(ctx, id)
        if err != nil {
            return err
        }

        _, err = s.repo.DeductQuantity(ctx, id, qty, inv.Version)
        if err == ErrOptimisticLock {
            continue // 재시도
        }
        return err
    }

    return ErrMaxRetriesExceeded
}
```

### 6.2 CHECK 제약 조건

```sql
-- 재고: 음수 방지, 예약 초과 방지
CONSTRAINT chk_quantity CHECK (quantity >= 0)
CONSTRAINT chk_reserved CHECK (reserved_quantity >= 0)
CONSTRAINT chk_available CHECK (quantity >= reserved_quantity)

-- 계정: 잔액 음수 방지
CONSTRAINT chk_balance CHECK (balance >= 0)
CONSTRAINT chk_hold CHECK (hold_balance >= 0)

-- 원장: 금액 양수 강제
CONSTRAINT chk_amount CHECK (amount > 0)
```

### 6.3 UNIQUE 제약 조건

```sql
-- 상품: SKU 중복 방지
UNIQUE KEY (sku) ON products

-- 재고: 동일 상품/위치 중복 방지
UNIQUE KEY uk_product_location (product_id, location) ON inventories

-- 결제: 멱등성 키 중복 방지
UNIQUE KEY (idempotency_key) ON payments
```

---

## 7. 분산락 (Distributed Lock)

### 7.1 필요성

Optimistic Lock만으로는 해결할 수 없는 시나리오:

1. **긴 트랜잭션**: 재고 조회 → 외부 API 호출 → 재고 업데이트
2. **복합 리소스 잠금**: 여러 재고를 동시에 잠가야 하는 경우
3. **재시도 비용**: Optimistic Lock 실패 시 재시도 비용이 큰 경우

### 7.2 Redis 기반 분산락 구현

**Redlock 알고리즘 단순화 버전:**

```go
// pkg/lock/distributed.go
package lock

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
)

var (
    ErrLockNotAcquired = errors.New("failed to acquire lock")
    ErrLockNotHeld     = errors.New("lock not held by this owner")
)

type DistributedLock struct {
    client *redis.Client
    key    string
    owner  string
    ttl    time.Duration
}

// NewLock creates a new distributed lock instance
func NewLock(client *redis.Client, resource string, ttl time.Duration) *DistributedLock {
    return &DistributedLock{
        client: client,
        key:    "lock:" + resource,
        owner:  uuid.New().String(),
        ttl:    ttl,
    }
}

// Acquire attempts to acquire the lock
// Returns error if lock is already held
func (l *DistributedLock) Acquire(ctx context.Context) error {
    // SET key owner NX PX ttl
    // NX: 키가 없을 때만 설정
    // PX: 밀리초 단위 TTL
    success, err := l.client.SetNX(ctx, l.key, l.owner, l.ttl).Result()
    if err != nil {
        return err
    }
    if !success {
        return ErrLockNotAcquired
    }
    return nil
}

// Release releases the lock
// Only releases if the lock is held by this owner
func (l *DistributedLock) Release(ctx context.Context) error {
    // Lua script for atomic check-and-delete
    // 내 owner가 아니면 삭제하지 않음 (다른 프로세스가 획득한 락 보호)
    script := redis.NewScript(`
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `)

    result, err := script.Run(ctx, l.client, []string{l.key}, l.owner).Int()
    if err != nil {
        return err
    }
    if result == 0 {
        return ErrLockNotHeld
    }
    return nil
}

// Extend extends the lock TTL
// Useful for long-running operations
func (l *DistributedLock) Extend(ctx context.Context, ttl time.Duration) error {
    script := redis.NewScript(`
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("pexpire", KEYS[1], ARGV[2])
        else
            return 0
        end
    `)

    result, err := script.Run(ctx, l.client, []string{l.key}, l.owner, ttl.Milliseconds()).Int()
    if err != nil {
        return err
    }
    if result == 0 {
        return ErrLockNotHeld
    }
    return nil
}
```

### 7.3 락 사용 패턴

**패턴 1: 단일 리소스 락**

```go
// 재고 예약 시 분산락 사용
func (s *InventoryService) Reserve(ctx context.Context, inventoryID int64, qty int64) error {
    // 1. 분산락 획득
    lock := lock.NewLock(s.redis, fmt.Sprintf("inventory:%d", inventoryID), 30*time.Second)

    if err := lock.Acquire(ctx); err != nil {
        return errors.LockFailed(fmt.Sprintf("inventory:%d", inventoryID))
    }
    defer lock.Release(ctx)

    // 2. 비즈니스 로직 (락 보호 하에 실행)
    inv, err := s.repo.Get(ctx, inventoryID)
    if err != nil {
        return err
    }

    available := inv.Quantity - inv.ReservedQuantity
    if available < qty {
        return errors.InsufficientStock(available, qty)
    }

    // 3. DB 업데이트 (여전히 Optimistic Lock도 사용 - 방어적)
    return s.repo.Reserve(ctx, inventoryID, qty, inv.Version)
}
```

**패턴 2: 복합 리소스 락**

```go
// 주문 생성 시 여러 상품 재고 동시 락
func (s *OrderService) Create(ctx context.Context, req CreateOrderRequest) error {
    // 1. 모든 재고에 대해 락 획득 (데드락 방지: ID 정렬)
    inventoryIDs := sortedInventoryIDs(req.Items)
    locks := make([]*lock.DistributedLock, len(inventoryIDs))

    for i, id := range inventoryIDs {
        locks[i] = lock.NewLock(s.redis, fmt.Sprintf("inventory:%d", id), 30*time.Second)
        if err := locks[i].Acquire(ctx); err != nil {
            // 이전에 획득한 락 해제
            for j := 0; j < i; j++ {
                locks[j].Release(ctx)
            }
            return err
        }
    }
    defer func() {
        for _, l := range locks {
            l.Release(ctx)
        }
    }()

    // 2. 트랜잭션 내에서 처리
    return s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
        // 주문 생성, 재고 예약 등
    })
}
```

### 7.4 분산락 주의사항

| 위험 | 설명 | 대응 |
|------|------|------|
| **락 만료** | 작업 중 락 TTL 만료 | Extend 주기적 호출 또는 충분한 TTL |
| **데드락** | 여러 락 순환 대기 | 락 획득 순서 정렬 (ID 오름차순) |
| **Redis 장애** | Single point of failure | Redlock (다중 Redis) 또는 장애 허용 설계 |
| **락 누수** | Release 실패 | defer + TTL 만료 의존 |

### 7.5 분산락 vs Optimistic Lock 선택 기준

| 상황 | 권장 방식 | 이유 |
|------|----------|------|
| 짧은 트랜잭션, 높은 동시성 | Optimistic Lock | 락 오버헤드 없음, 충돌 시 재시도 |
| 긴 트랜잭션, 외부 호출 포함 | 분산락 | 재시도 비용 방지 |
| 여러 리소스 동시 잠금 | 분산락 | 원자적 다중 리소스 보호 |
| 조회 후 조건부 업데이트 | 둘 다 사용 | 분산락 + Optimistic Lock (방어적) |

---

## 8. Double-Entry Ledger

### 8.1 기본 원칙

**복식부기 규칙:**
- 모든 거래는 **차변(DEBIT)**과 **대변(CREDIT)**으로 기록
- 하나의 tx_id에서 **SUM(DEBIT) = SUM(CREDIT)** 강제
- 원장은 **불변(INSERT ONLY)** - 수정/삭제 불가

### 8.2 계정 유형별 차대변 의미

| 계정 유형 | DEBIT 의미 | CREDIT 의미 |
|----------|-----------|-------------|
| USER | 잔액 감소 (출금/홀드) | 잔액 증가 (입금) |
| MERCHANT | 잔액 감소 | 잔액 증가 (정산 수취) |
| ESCROW | 예치금 출금 | 예치금 입금 |
| SYSTEM | 수수료 지출 | 수수료 수취 |

### 8.3 거래별 원장 기록 예시

**예시 1: 결제 승인 (Authorization) - 100 KRWS 홀드**

```
┌─────────────────────────────────────────────────────────────────┐
│ tx_id: "auth_20240115_001"                                      │
├──────────────┬────────────┬────────┬────────────────────────────┤
│ account_id   │ entry_type │ amount │ 설명                       │
├──────────────┼────────────┼────────┼────────────────────────────┤
│ USER_A (1)   │ DEBIT      │ 100    │ 가용잔액 → 홀드 전환       │
│ ESCROW (99)  │ CREDIT     │ 100    │ 에스크로 예치              │
├──────────────┴────────────┴────────┴────────────────────────────┤
│ 검증: DEBIT(100) = CREDIT(100) ✅                              │
├─────────────────────────────────────────────────────────────────┤
│ accounts 변경:                                                  │
│   USER_A: balance 100→100 (변동 없음), hold_balance 0→100      │
│   ESCROW: balance 0→100                                         │
└─────────────────────────────────────────────────────────────────┘
```

**예시 2: 결제 캡처 + 정산 (수수료 3%)**

```
┌─────────────────────────────────────────────────────────────────┐
│ tx_id: "capture_20240115_001"                                   │
├──────────────┬────────────┬────────┬────────────────────────────┤
│ account_id   │ entry_type │ amount │ 설명                       │
├──────────────┼────────────┼────────┼────────────────────────────┤
│ USER_A (1)   │ CREDIT     │ 100    │ 홀드 해제 (balance 차감)   │
│ ESCROW (99)  │ DEBIT      │ 100    │ 에스크로 출금              │
│ MERCHANT (2) │ CREDIT     │ 97     │ 판매자 정산 (97%)          │
│ SYSTEM (100) │ CREDIT     │ 3      │ 플랫폼 수수료 (3%)         │
├──────────────┴────────────┴────────┴────────────────────────────┤
│ 검증: DEBIT(100) = CREDIT(100+97+3... 아니다!)                 │
└─────────────────────────────────────────────────────────────────┘
```

**수정된 예시 2:**

```
┌─────────────────────────────────────────────────────────────────┐
│ tx_id: "capture_20240115_001"                                   │
├──────────────┬────────────┬────────┬────────────────────────────┤
│ account_id   │ entry_type │ amount │ 설명                       │
├──────────────┼────────────┼────────┼────────────────────────────┤
│ ESCROW (99)  │ DEBIT      │ 100    │ 에스크로 출금              │
│ MERCHANT (2) │ CREDIT     │ 97     │ 판매자 정산 (97%)          │
│ SYSTEM (100) │ CREDIT     │ 3      │ 플랫폼 수수료 (3%)         │
├──────────────┴────────────┴────────┴────────────────────────────┤
│ 검증: DEBIT(100) = CREDIT(97+3=100) ✅                         │
├─────────────────────────────────────────────────────────────────┤
│ accounts 변경:                                                  │
│   USER_A: balance 100→0, hold_balance 100→0                    │
│   ESCROW: balance 100→0                                         │
│   MERCHANT: balance 0→97                                        │
│   SYSTEM: balance 0→3                                           │
└─────────────────────────────────────────────────────────────────┘
```

**예시 3: 환불 (Refund)**

```
┌─────────────────────────────────────────────────────────────────┐
│ tx_id: "refund_20240116_001"                                    │
├──────────────┬────────────┬────────┬────────────────────────────┤
│ account_id   │ entry_type │ amount │ 설명                       │
├──────────────┼────────────┼────────┼────────────────────────────┤
│ MERCHANT (2) │ DEBIT      │ 97     │ 판매자 환불 출금           │
│ SYSTEM (100) │ DEBIT      │ 3      │ 수수료 환불 (선택적)       │
│ USER_A (1)   │ CREDIT     │ 100    │ 구매자 환불 입금           │
├──────────────┴────────────┴────────┴────────────────────────────┤
│ 검증: DEBIT(97+3=100) = CREDIT(100) ✅                         │
└─────────────────────────────────────────────────────────────────┘
```

### 8.4 Go 구현

```go
// internal/ledger/service.go
package ledger

type Entry struct {
    AccountID     int64
    EntryType     string // "DEBIT" or "CREDIT"
    Amount        decimal.Decimal
    ReferenceType string
    ReferenceID   int64
    Description   string
}

type Service struct {
    repo *Repository
}

// Post records a balanced ledger transaction
func (s *Service) Post(ctx context.Context, txID string, entries []Entry) error {
    // 1. 차대변 검증
    var debitSum, creditSum decimal.Decimal
    for _, e := range entries {
        if e.Amount.LessThanOrEqual(decimal.Zero) {
            return errors.InvalidInput("amount must be positive")
        }
        switch e.EntryType {
        case "DEBIT":
            debitSum = debitSum.Add(e.Amount)
        case "CREDIT":
            creditSum = creditSum.Add(e.Amount)
        default:
            return errors.InvalidInput("invalid entry type")
        }
    }

    if !debitSum.Equal(creditSum) {
        return errors.InvalidInput(fmt.Sprintf(
            "unbalanced transaction: debit=%s, credit=%s",
            debitSum.String(), creditSum.String(),
        ))
    }

    // 2. 트랜잭션 내에서 기록
    return s.repo.WithTransaction(ctx, func(tx *sql.Tx) error {
        for _, e := range entries {
            // 계정 잔액 업데이트
            if err := s.updateAccountBalance(ctx, tx, e); err != nil {
                return err
            }

            // 원장 기록
            if err := s.repo.InsertEntry(ctx, tx, txID, e); err != nil {
                return err
            }
        }
        return nil
    })
}

func (s *Service) updateAccountBalance(ctx context.Context, tx *sql.Tx, e Entry) error {
    switch e.EntryType {
    case "DEBIT":
        // 잔액 감소 (출금)
        return s.repo.DebitAccount(ctx, tx, e.AccountID, e.Amount)
    case "CREDIT":
        // 잔액 증가 (입금)
        return s.repo.CreditAccount(ctx, tx, e.AccountID, e.Amount)
    }
    return nil
}
```

---

## 9. Outbox 패턴

### 9.1 필요성

**문제:**
```
BEGIN TX
  INSERT INTO payments (...);          -- 성공
  CALL external_notification_api();    -- 실패 → 롤백
COMMIT
```

외부 API 호출이 트랜잭션에 포함되면:
- 외부 API 실패 시 전체 롤백
- 외부 API 지연 시 DB 락 장시간 점유
- 외부 API 성공했는데 이후 로직 실패 시 일관성 깨짐

**해결: Transactional Outbox**
```
BEGIN TX
  INSERT INTO payments (...);          -- 비즈니스 데이터
  INSERT INTO outbox (...);            -- 이벤트 (같은 TX)
COMMIT

-- 별도 Worker가 outbox 폴링 후 외부 API 호출
```

### 9.2 Outbox 흐름

```
┌─────────────────────────────────────────────────────────────────┐
│                    Transactional Outbox Pattern                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐                                                │
│  │ API Handler │                                                │
│  └──────┬──────┘                                                │
│         │                                                        │
│         ▼                                                        │
│  ┌─────────────────────────────────────┐                        │
│  │           BEGIN TRANSACTION          │                        │
│  │  ┌─────────────────────────────────┐│                        │
│  │  │ INSERT INTO payments (...)      ││ ← 비즈니스 데이터      │
│  │  └─────────────────────────────────┘│                        │
│  │  ┌─────────────────────────────────┐│                        │
│  │  │ INSERT INTO outbox (            ││                        │
│  │  │   event_type='PAYMENT_CREATED', ││ ← 이벤트              │
│  │  │   payload={...}                 ││                        │
│  │  │ )                               ││                        │
│  │  └─────────────────────────────────┘│                        │
│  │           COMMIT                     │                        │
│  └─────────────────────────────────────┘                        │
│                    │                                             │
│                    │ 원자적 저장 보장                            │
│                    ▼                                             │
│  ┌─────────────────────────────────────┐                        │
│  │           Outbox Worker              │ (별도 프로세스)       │
│  │  ┌─────────────────────────────────┐│                        │
│  │  │ SELECT * FROM outbox            ││                        │
│  │  │ WHERE status = 'PENDING'        ││                        │
│  │  └─────────────────────────────────┘│                        │
│  │                 │                    │                        │
│  │                 ▼                    │                        │
│  │  ┌─────────────────────────────────┐│                        │
│  │  │ UPDATE status = 'PROCESSING'    ││ ← 락 (중복 처리 방지) │
│  │  └─────────────────────────────────┘│                        │
│  │                 │                    │                        │
│  │                 ▼                    │                        │
│  │  ┌─────────────────────────────────┐│                        │
│  │  │ 외부 시스템 호출                 ││                        │
│  │  │ (알림, 체인, 웹훅 등)           ││                        │
│  │  └─────────────────────────────────┘│                        │
│  │                 │                    │                        │
│  │        성공     │     실패           │                        │
│  │                 ▼                    │                        │
│  │  ┌──────────────────────────────────┐│                        │
│  │  │ status='COMPLETED'   │ 'FAILED'  ││                        │
│  │  │                      │ retry++   ││                        │
│  │  └──────────────────────────────────┘│                        │
│  └─────────────────────────────────────┘                        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 9.3 Worker 구현

```go
// internal/outbox/worker.go
package outbox

type Worker struct {
    repo       *Repository
    processors map[string]Processor
    interval   time.Duration
    batchSize  int
}

type Processor interface {
    Process(ctx context.Context, event *OutboxEvent) error
}

func (w *Worker) Run(ctx context.Context) error {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := w.processBatch(ctx); err != nil {
                log.Error("batch processing failed", zap.Error(err))
            }
        }
    }
}

func (w *Worker) processBatch(ctx context.Context) error {
    // 1. 처리할 이벤트 조회 + 락
    events, err := w.repo.FetchAndLock(ctx, w.batchSize)
    if err != nil {
        return err
    }

    for _, event := range events {
        if err := w.processEvent(ctx, event); err != nil {
            log.Error("event processing failed",
                zap.Int64("event_id", event.ID),
                zap.Error(err),
            )
        }
    }

    return nil
}

func (w *Worker) processEvent(ctx context.Context, event *OutboxEvent) error {
    processor, ok := w.processors[event.EventType]
    if !ok {
        return w.repo.MarkDeadLetter(ctx, event.ID, "unknown event type")
    }

    // 처리 시도
    if err := processor.Process(ctx, event); err != nil {
        return w.handleFailure(ctx, event, err)
    }

    // 성공
    return w.repo.MarkCompleted(ctx, event.ID)
}

func (w *Worker) handleFailure(ctx context.Context, event *OutboxEvent, err error) error {
    event.RetryCount++

    if event.RetryCount >= event.MaxRetries {
        return w.repo.MarkDeadLetter(ctx, event.ID, err.Error())
    }

    // Exponential backoff
    nextRetry := time.Now().Add(w.calculateBackoff(event.RetryCount))
    return w.repo.MarkFailed(ctx, event.ID, err.Error(), nextRetry)
}

func (w *Worker) calculateBackoff(retryCount int) time.Duration {
    // 1s, 2s, 4s, 8s, 16s...
    return time.Duration(1<<retryCount) * time.Second
}
```

### 9.4 재시도 전략 (Exponential Backoff)

```
retry_count │ next_retry_at (delay)
────────────┼──────────────────────
     0      │ 즉시
     1      │ +1초
     2      │ +2초
     3      │ +4초
     4      │ +8초
     5      │ DEAD_LETTER (수동 처리)
```

---

## 10. 트랜잭션 경계

### 10.1 유스케이스별 트랜잭션 범위

| 유스케이스 | 트랜잭션 범위 | 이유 |
|------------|--------------|------|
| **주문 생성** | orders + order_items + inventories (reserve) + outbox | 재고 예약과 주문이 원자적이어야 함 |
| **주문 확정** | orders + inventories (deduct) + inventory_logs | 재고 차감과 이력이 원자적이어야 함 |
| **결제 승인** | payments + accounts + ledger_entries + outbox | 잔액 홀드와 원장 기록이 원자적이어야 함 |
| **결제 캡처** | payments + settlements + accounts + ledger_entries | 정산과 원장 기록이 원자적이어야 함 |
| **환불** | payments + accounts + ledger_entries + outbox | 잔액 복구와 원장 기록이 원자적이어야 함 |

### 10.2 트랜잭션 패턴 예시

```go
// internal/order/service.go
func (s *OrderService) Create(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    // 1. 분산락 획득 (재고)
    locks := s.acquireInventoryLocks(ctx, req.Items)
    defer s.releaseLocks(locks)

    var order *Order

    // 2. DB 트랜잭션
    err := s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
        // 2.1 주문 생성
        order, err = s.orderRepo.Create(ctx, tx, req)
        if err != nil {
            return err
        }

        // 2.2 주문 상품 생성
        for _, item := range req.Items {
            if err := s.orderItemRepo.Create(ctx, tx, order.ID, item); err != nil {
                return err
            }
        }

        // 2.3 재고 예약
        for _, item := range req.Items {
            if err := s.inventoryRepo.Reserve(ctx, tx, item.InventoryID, item.Quantity); err != nil {
                return err
            }
        }

        // 2.4 Outbox 이벤트 생성
        event := OutboxEvent{
            EventType:     "ORDER_CREATED",
            AggregateType: "order",
            AggregateID:   order.ID,
            Payload:       order,
        }
        if err := s.outboxRepo.Create(ctx, tx, event); err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return order, nil
}
```

---

## 11. API 설계

### 11.1 Phase 1: 상품/재고

```
POST   /api/v1/products              # 상품 생성
GET    /api/v1/products/:id          # 상품 조회
GET    /api/v1/products              # 상품 목록
PATCH  /api/v1/products/:id          # 상품 수정

POST   /api/v1/inventory/inbound     # 입고
POST   /api/v1/inventory/outbound    # 출고
GET    /api/v1/inventory/:sku        # 재고 조회
GET    /api/v1/inventory/:sku/logs   # 재고 이력
```

### 11.2 Phase 2: 주문

```
POST   /api/v1/orders                # 주문 생성
GET    /api/v1/orders/:id            # 주문 조회
GET    /api/v1/orders                # 주문 목록
PATCH  /api/v1/orders/:id/confirm    # 주문 확정
PATCH  /api/v1/orders/:id/cancel     # 주문 취소
PATCH  /api/v1/orders/:id/ship       # 배송 시작
PATCH  /api/v1/orders/:id/complete   # 주문 완료
```

### 11.3 Phase 3: 결제/정산

```
POST   /api/v1/payments/authorize    # 결제 승인
POST   /api/v1/payments/capture      # 결제 캡처
POST   /api/v1/payments/void         # 승인 취소
POST   /api/v1/payments/refund       # 환불

POST   /api/v1/settlements/execute   # 정산 실행
GET    /api/v1/settlements/:id       # 정산 조회

GET    /api/v1/accounts/:id/balance  # 잔액 조회
GET    /api/v1/accounts/:id/ledger   # 원장 조회
```

### 11.4 공통 헤더

| 헤더 | 용도 | 예시 |
|------|------|------|
| `X-Request-ID` | 요청 추적 | `req-abc123` |
| `X-Idempotency-Key` | 멱등성 보장 | `order-create-xyz` |
| `Authorization` | 인증 | `Bearer <token>` |

---

## 12. 에러 처리

### 12.1 표준 에러 코드

```go
// 4xx Client Errors
const (
    CodeInvalidInput        = "INVALID_INPUT"
    CodeNotFound            = "NOT_FOUND"
    CodeConflict            = "CONFLICT"
    CodeIdempotencyConflict = "IDEMPOTENCY_CONFLICT"
    CodeInsufficientBalance = "INSUFFICIENT_BALANCE"
    CodeInsufficientStock   = "INSUFFICIENT_STOCK"
    CodeInvalidState        = "INVALID_STATE_TRANSITION"
)

// 5xx Server Errors
const (
    CodeInternal     = "INTERNAL_ERROR"
    CodeDBError      = "DB_ERROR"
    CodeLockFailed   = "LOCK_FAILED"
    CodeChainError   = "CHAIN_ERROR"
)
```

### 12.2 에러 응답 형식

```json
{
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Available stock 50 is less than requested 100",
    "request_id": "req-abc123",
    "details": {
      "available": 50,
      "requested": 100,
      "inventory_id": 42
    }
  }
}
```

---

## 부록: 검증 명령어

```bash
# 서비스 시작
make up              # docker-compose 기동
make migrate         # DB 마이그레이션
make run             # API 서버 실행

# 검증
curl http://localhost:8080/health
curl http://localhost:8080/ready
open http://localhost:8080/swagger/index.html

# 코드 생성
make sqlc            # sqlc 쿼리 생성
make swag            # Swagger 문서 생성
make generate        # 전체 생성
```

---

**문서 버전:** 1.0
**최종 수정:** 2024-01-23
**작성자:** Claude (시니어 Go 백엔드 엔지니어)

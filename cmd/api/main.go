package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/docs"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/common/handler"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/common/middleware"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/config"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/user"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/wallet"
	pkgdb "github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/pkg/db"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/pkg/eip712"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/pkg/nonce"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// @title B2B Commerce Settlement Engine API
// @version 1.0
// @description 재고관리 + 주문처리 + 스테이블코인 정산 API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

func main() {
	// 1) 로거 초기화
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 2) 설정 로드
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	logger.Info("starting server",
		zap.String("environment", cfg.Server.Environment),
		zap.String("addr", cfg.Server.Addr()),
	)

	// 3) DB 초기화
	db, err := initDB(cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// 4) Redis 초기화
	rdb := initRedis(cfg.Redis)
	defer rdb.Close()

	// 5) 연결 테스트 (fail-fast)
	if err := testConnections(db, rdb); err != nil {
		logger.Fatal("failed to test connections", zap.Error(err))
	}

	// 6) 라우터 구성
	router := setupRouter(cfg, logger, db, rdb)

	// 7) HTTP 서버 생성
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 8) 서버 비동기 시작
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	logger.Info("server started",
		zap.String("addr", cfg.Server.Addr()),
		zap.String("swagger", fmt.Sprintf("http://localhost:%d/swagger/index.html", cfg.Server.Port)),
	)

	// 9) 종료 시그널 대기
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// 10) Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}

func initLogger() (*zap.Logger, error) {
	env := os.Getenv("ENVIRONMENT")
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func initDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return db, nil
}

func initRedis(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

func testConnections(db *sql.DB, rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

func setupRouter(cfg *config.Config, logger *zap.Logger, db *sql.DB, rdb *redis.Client) *gin.Engine {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))

	// Swagger 설정
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.Server.Port)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health endpoints
	healthHandler := handler.NewHealthHandler(db, rdb)
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	// ============================================================================
	// Dependencies Setup
	// ============================================================================

	// TxRunner for transaction management
	txRunner := pkgdb.NewTxRunner(db)

	// Nonce store for EIP-712 replay protection
	nonceStore := nonce.NewRedisStore(rdb, logger)

	// EIP-712 verifier for wallet signature verification
	verifier := eip712.NewEthVerifier(eip712.Config{
		ChainID:            cfg.EIP712.ChainID,
		VerifyingContract:  cfg.EIP712.VerifyingContract,
		TimestampTolerance: cfg.EIP712.TimestampTolerance,
	}, nonceStore, logger)

	// ============================================================================
	// Service & Handler Setup
	// ============================================================================

	// User service & handler
	userService := user.NewService(txRunner, logger)
	userHandler := user.NewHandler(userService)

	// Wallet service & handler
	walletService := wallet.NewService(txRunner, verifier, logger)
	walletHandler := wallet.NewHandler(walletService)

	// ============================================================================
	// Route Registration
	// ============================================================================

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Phase 1: User & Wallet
		userHandler.RegisterRoutes(v1)
		walletHandler.RegisterRoutes(v1)

		// Phase 2: Products & Inventory (TODO)
		_ = v1.Group("/products")
		_ = v1.Group("/inventory")

		// Phase 3: Orders (TODO)
		_ = v1.Group("/orders")

		// Phase 4: Payments & Settlements (TODO)
		_ = v1.Group("/payments")
		_ = v1.Group("/settlements")
		_ = v1.Group("/accounts")
	}

	return router
}

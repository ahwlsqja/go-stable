package nonce

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// keyPrefix is the Redis key prefix for nonces
	keyPrefix = "nonce"
)

// RedisStore implements Store interface using Redis
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
	logger *zap.Logger
}

// Compile-time interface compliance check
var _ Store = (*RedisStore)(nil)

// NewRedisStore creates a new Redis-based nonce store with default TTL
func NewRedisStore(client *redis.Client, logger *zap.Logger) *RedisStore {
	return &RedisStore{
		client: client,
		ttl:    DefaultTTL,
		logger: logger,
	}
}

// NewRedisStoreWithTTL creates a new Redis-based nonce store with custom TTL
func NewRedisStoreWithTTL(client *redis.Client, ttl time.Duration, logger *zap.Logger) *RedisStore {
	return &RedisStore{
		client: client,
		ttl:    ttl,
		logger: logger,
	}
}

// buildKey creates a Redis key from address and nonce
// Format: nonce:{lowercase_address}:{nonce}
func buildKey(address, nonce string) string {
	return fmt.Sprintf("%s:%s:%s", keyPrefix, strings.ToLower(address), nonce)
}

// Reserve attempts to reserve a nonce using SETNX
func (s *RedisStore) Reserve(ctx context.Context, nonce, address string) error {
	key := buildKey(address, nonce)

	// SETNX with TTL - only succeeds if key doesn't exist
	ok, err := s.client.SetNX(ctx, key, "reserved", s.ttl).Result()
	if err != nil {
		s.logger.Error("failed to reserve nonce",
			zap.String("address", address),
			zap.String("nonce", nonce),
			zap.Error(err),
		)
		return fmt.Errorf("failed to reserve nonce: %w", err)
	}

	if !ok {
		s.logger.Warn("nonce already used or reserved",
			zap.String("address", address),
			zap.String("nonce", nonce),
		)
		return ErrNonceAlreadyUsed
	}

	s.logger.Debug("nonce reserved",
		zap.String("address", address),
		zap.String("nonce", nonce),
	)
	return nil
}

// MarkUsed marks a reserved nonce as used
func (s *RedisStore) MarkUsed(ctx context.Context, nonce, address string) error {
	key := buildKey(address, nonce)

	err := s.client.Set(ctx, key, "used", s.ttl).Err()
	if err != nil {
		s.logger.Error("failed to mark nonce as used",
			zap.String("address", address),
			zap.String("nonce", nonce),
			zap.Error(err),
		)
		return fmt.Errorf("failed to mark nonce as used: %w", err)
	}

	s.logger.Debug("nonce marked as used",
		zap.String("address", address),
		zap.String("nonce", nonce),
	)
	return nil
}

// Release releases a reserved nonce, allowing retry
func (s *RedisStore) Release(ctx context.Context, nonce, address string) error {
	key := buildKey(address, nonce)

	err := s.client.Del(ctx, key).Err()
	if err != nil {
		s.logger.Error("failed to release nonce",
			zap.String("address", address),
			zap.String("nonce", nonce),
			zap.Error(err),
		)
		return fmt.Errorf("failed to release nonce: %w", err)
	}

	s.logger.Debug("nonce released",
		zap.String("address", address),
		zap.String("nonce", nonce),
	)
	return nil
}

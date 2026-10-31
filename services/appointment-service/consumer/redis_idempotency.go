package consumer

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	idemPrefix    = "medcore:idem:pharmacy:"
	processingTTL = 5 * time.Minute
	successTTL    = 24 * time.Hour
)

type RedisIdempotency struct {
	rdb *redis.Client
}

func NewRedisIdempotency(addr string) *RedisIdempotency {
	return &RedisIdempotency{
		rdb: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

func (r *RedisIdempotency) key(eventID string) string {
	return idemPrefix + eventID
}

// TryAcquire mengembalikan status dan apakah key berhasil diakuisisi.
func (r *RedisIdempotency) TryAcquire(ctx context.Context, eventID string) (string, bool, error) {
	key := r.key(eventID)

	// Coba set key dengan status PROCESSING
	ok, err := r.rdb.SetNX(ctx, key, "PROCESSING", processingTTL).Result()
	if err != nil {
		return "", false, err
	}
	if ok {
		return "PROCESSING", true, nil
	}

	// Jika gagal set, cek status saat ini
	status, err := r.rdb.Get(ctx, key).Result()
	
	// Edge case: key expired tepat setelah SetNX gagal, coba ambil alih
	if errors.Is(err, redis.Nil) {
		ok2, err2 := r.rdb.SetNX(ctx, key, "PROCESSING", processingTTL).Result()
		if err2 != nil {
			return "", false, err2
		}
		if ok2 {
			return "PROCESSING", true, nil
		}
		status, err = r.rdb.Get(ctx, key).Result()
	}
	
	if err != nil {
		return "", false, err
	}
	
	return status, false, nil
}

// MarkSuccess menandai bahwa event telah selesai diproses
func (r *RedisIdempotency) MarkSuccess(ctx context.Context, eventID string) error {
	return r.rdb.Set(ctx, r.key(eventID), "SUCCESS", successTTL).Err()
}

// Release menghapus key (digunakan jika proses gagal agar bisa di-retry)
func (r *RedisIdempotency) Release(ctx context.Context, eventID string) {
	_ = r.rdb.Del(ctx, r.key(eventID)).Err()
}
package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"maintify/core/pkg/config"
	"maintify/core/pkg/logger"
	"strings"

	"github.com/redis/go-redis/v9"
)

type HookFunc func(payload string)

var (
	rdb *redis.Client
	ctx = context.Background()
	// For testing
	initializeFunc = Initialize
	newClientFunc  = redis.NewClient
	fatalfFunc     = func(msg string, err error) {
		logger.Fatal(msg, err)
	}
)

// Init initializes the hooks system and exits on error
func Init() {
	if err := initializeFunc(); err != nil {
		fatalfFunc("Failed to initialize hooks", err)
	}
}

// Initialize sets up the hooks system
func Initialize() error {
	// Handle Redis URL format - if it's just host:port, convert to redis:// format
	if config.Current == nil {
		return fmt.Errorf("config not loaded")
	}

	redisURL := config.Current.RedisURL
	if redisURL == "" {
		redisURL = "redis:6379"
	}
	if !strings.HasPrefix(redisURL, "redis://") {
		redisURL = "redis://" + redisURL
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("could not parse Redis URL '%s': %w", redisURL, err)
	}

	client := newClientFunc(opt)
	return InitWithClient(client)
}

// InitWithClient initializes the hooks system with a provided Redis client
func InitWithClient(client *redis.Client) error {
	rdb = client

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("could not connect to Redis: %w", err)
	}
	logger.Info("Successfully connected to Redis for hooks system.")
	return nil
}

func RegisterHook(event string, fn HookFunc) {
	if rdb == nil {
		logger.Warn(fmt.Sprintf("Redis client not initialized, cannot register hook for '%s'", event))
		return
	}
	// Capture client to avoid race condition if rdb is modified (e.g. in tests)
	client := rdb
	// Capture context to avoid race condition if ctx is modified (e.g. in tests)
	currentCtx := ctx
	go func() {
		pubsub := client.Subscribe(currentCtx, event)
		defer pubsub.Close()

		ch := pubsub.Channel()
		processMessages(currentCtx, event, ch, fn)
	}()
}

func processMessages(ctx context.Context, event string, ch <-chan *redis.Message, fn HookFunc) {
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			logger.Info(fmt.Sprintf("Received hook for event '%s'", event))
			fn(msg.Payload)
		case <-ctx.Done():
			return
		}
	}
}

func Trigger(event string, payload interface{}) error {
	if rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to marshal hook payload for event '%s'", event), err)
		return err
	}

	err = rdb.Publish(ctx, event, payloadBytes).Err()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to publish hook for event '%s'", event), err)
		return err
	}
	logger.Info(fmt.Sprintf("Triggered hook for event '%s'", event))
	return nil
}

// Cleanup gracefully closes Redis connection
func Cleanup() {
	if rdb != nil {
		closeClient(rdb)
	}
}

type closer interface {
	Close() error
}

func closeClient(c closer) {
	if err := c.Close(); err != nil {
		logger.Error("Error closing Redis connection", err)
	} else {
		logger.Info("Redis connection closed gracefully")
	}
}

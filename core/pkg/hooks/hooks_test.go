package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"maintify/core/pkg/config"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupConfig() {
	config.Current = &config.Config{
		RedisURL: "redis://localhost:6379",
	}
}

func TestInit(t *testing.T) {
	// Save original and restore
	orig := initializeFunc
	defer func() { initializeFunc = orig }()

	t.Run("calls initializeFunc", func(t *testing.T) {
		called := false
		initializeFunc = func() error {
			called = true
			return nil
		}
		Init()
		assert.True(t, called)
	})

	t.Run("calls fatalf on error", func(t *testing.T) {
		origFatal := fatalfFunc
		defer func() { fatalfFunc = origFatal }()

		initializeFunc = func() error {
			return errors.New("init failed")
		}

		fatalCalled := false
		fatalfFunc = func(msg string, err error) {
			fatalCalled = true
		}

		Init()
		assert.True(t, fatalCalled)
	})
}

func TestInitialize(t *testing.T) {
	t.Run("returns error if config not loaded", func(t *testing.T) {
		config.Current = nil
		err := Initialize()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config not loaded")
	})

	t.Run("returns error for invalid redis url", func(t *testing.T) {
		config.Current = &config.Config{
			RedisURL: string([]byte{0x7f}),
		}
		err := Initialize()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not parse Redis URL")
	})

	t.Run("succeeds when client creation and ping succeed", func(t *testing.T) {
		setupConfig()
		orig := newClientFunc
		defer func() { newClientFunc = orig }()

		db, mock := redismock.NewClientMock()
		newClientFunc = func(opt *redis.Options) *redis.Client {
			return db
		}

		mock.ExpectPing().SetVal("PONG")

		err := Initialize()
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("uses default redis url if empty", func(t *testing.T) {
		config.Current = &config.Config{
			RedisURL: "",
		}
		orig := newClientFunc
		defer func() { newClientFunc = orig }()

		db, mock := redismock.NewClientMock()
		newClientFunc = func(opt *redis.Options) *redis.Client {
			assert.Equal(t, "redis:6379", opt.Addr)
			return db
		}
		mock.ExpectPing().SetVal("PONG")

		err := Initialize()
		assert.NoError(t, err)
	})

	t.Run("adds redis:// prefix if missing", func(t *testing.T) {
		config.Current = &config.Config{
			RedisURL: "localhost:6379",
		}
		orig := newClientFunc
		defer func() { newClientFunc = orig }()

		db, mock := redismock.NewClientMock()
		newClientFunc = func(opt *redis.Options) *redis.Client {
			assert.Equal(t, "localhost:6379", opt.Addr)
			return db
		}
		mock.ExpectPing().SetVal("PONG")

		err := Initialize()
		assert.NoError(t, err)
	})
}

func TestInitWithClient(t *testing.T) {
	db, mock := redismock.NewClientMock()

	t.Run("succeeds when ping succeeds", func(t *testing.T) {
		mock.ExpectPing().SetVal("PONG")
		err := InitWithClient(db)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when ping fails", func(t *testing.T) {
		mock.ExpectPing().SetErr(errors.New("connection failed"))
		err := InitWithClient(db)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not connect to Redis")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTrigger(t *testing.T) {
	db, mock := redismock.NewClientMock()
	// Setup global rdb
	rdb = db
	defer func() { rdb = nil }()

	t.Run("fails if redis not initialized", func(t *testing.T) {
		rdb = nil
		err := Trigger("event", "payload")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
	})

	t.Run("publishes message successfully", func(t *testing.T) {
		rdb = db
		payload := "test-payload"
		payloadBytes, _ := json.Marshal(payload)
		mock.ExpectPublish("test-event", payloadBytes).SetVal(1)

		err := Trigger("test-event", payload)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when publish fails", func(t *testing.T) {
		rdb = db
		payload := "test-payload"
		payloadBytes, _ := json.Marshal(payload)
		mock.ExpectPublish("test-event", payloadBytes).SetErr(errors.New("publish failed"))

		err := Trigger("test-event", payload)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publish failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fails when marshal fails", func(t *testing.T) {
		rdb = db
		// Channel that cannot be marshaled
		err := Trigger("test-event", make(chan int))
		assert.Error(t, err)
		// json.Marshal error
	})
}

func TestCleanup(t *testing.T) {
	db, mock := redismock.NewClientMock()
	rdb = db
	defer func() { rdb = nil }()

	t.Run("closes redis connection", func(t *testing.T) {
		// redismock doesn't support ExpectClose, so we just call it.
		// It shouldn't panic.
		Cleanup()
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("does nothing if rdb is nil", func(t *testing.T) {
		rdb = nil
		Cleanup()
		// Should not panic
	})
}

func TestRegisterHook(t *testing.T) {
	db, _ := redismock.NewClientMock()
	rdb = db
	defer func() { rdb = nil }()

	t.Run("does nothing if redis not initialized", func(t *testing.T) {
		rdb = nil
		// Should not panic
		RegisterHook("event", func(p string) {})
	})

	t.Run("subscribes to event", func(t *testing.T) {
		rdb = db
		// Use a cancellable context to ensure the goroutine exits
		c, cancel := context.WithCancel(context.Background())
		// Override the package-level ctx
		oldCtx := ctx
		ctx = c
		defer func() {
			ctx = oldCtx
			cancel()
		}()

		RegisterHook("test-event", func(p string) {})

		// Allow goroutine to start
		time.Sleep(10 * time.Millisecond)

		// Cancel context to stop the goroutine
		cancel()
		time.Sleep(10 * time.Millisecond)
	})
}

func TestProcessMessages(t *testing.T) {
	ch := make(chan *redis.Message)
	received := make(chan string)
	done := make(chan struct{})

	fn := func(payload string) {
		received <- payload
	}

	go func() {
		processMessages(context.Background(), "test-event", ch, fn)
		close(done)
	}()

	ch <- &redis.Message{Payload: "test-payload"}

	select {
	case msg := <-received:
		assert.Equal(t, "test-payload", msg)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for message")
	}

	close(ch)

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for processMessages to exit")
	}
}

type mockCloser struct {
	err error
}

func (m *mockCloser) Close() error { return m.err }

func TestCloseClient(t *testing.T) {
	t.Run("logs success", func(t *testing.T) {
		closeClient(&mockCloser{err: nil})
	})
	t.Run("logs error", func(t *testing.T) {
		closeClient(&mockCloser{err: errors.New("close failed")})
	})
}

func TestProcessMessages_ContextCancellation(t *testing.T) {
	ch := make(chan *redis.Message)
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	fn := func(payload string) {}

	go func() {
		processMessages(ctx, "test-event", ch, fn)
		close(done)
	}()

	// Cancel context immediately
	cancel()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for processMessages to exit on context cancellation")
	}
}

func TestDefaultFatalFunc(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		fatalfFunc("test fatal", errors.New("test error"))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestDefaultFatalFunc")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

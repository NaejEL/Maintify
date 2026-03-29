package rbac

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTimeBasedAccessProcessor_Process(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockRBACService)
		processor := NewTimeBasedAccessProcessor(mockService)

		mockService.On("ProcessScheduledActivations", mock.Anything).Return(nil)
		mockService.On("CleanupExpiredAssignments", mock.Anything).Return(nil)

		processor.processTimeBasedAccess()
		mockService.AssertExpectations(t)
	})
}

func TestEmergencyAccessProcessor_Process(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockRBACService)
		processor := NewEmergencyAccessProcessor(mockService)

		mockService.On("ProcessEmergencyAccessEscalations", mock.Anything).Return(nil)

		err := processor.processEscalations(context.Background())
		assert.NoError(t, err)
		mockService.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockService := new(MockRBACService)
		processor := NewEmergencyAccessProcessor(mockService)

		mockService.On("ProcessEmergencyAccessEscalations", mock.Anything).Return(errors.New("escalation error"))

		err := processor.processEscalations(context.Background())
		assert.Error(t, err)
		mockService.AssertExpectations(t)
	})
}

func TestEmergencyAccessProcessor_Start(t *testing.T) {
	t.Run("Context Cancelled", func(t *testing.T) {
		mockService := new(MockRBACService)
		processor := NewEmergencyAccessProcessor(mockService)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Should return immediately
		processor.Start(ctx)
	})

	t.Run("Stop Called", func(t *testing.T) {
		mockService := new(MockRBACService)
		processor := NewEmergencyAccessProcessor(mockService)

		go func() {
			time.Sleep(10 * time.Millisecond)
			processor.Stop()
		}()

		processor.Start(context.Background())
	})
}

func TestEmergencyAccessProcessor_Run(t *testing.T) {
	t.Run("Process Tickers", func(t *testing.T) {
		mockService := new(MockRBACService)
		processor := NewEmergencyAccessProcessor(mockService)

		// Expect calls
		mockService.On("ProcessEmergencyAccessEscalations", mock.Anything).Return(nil).Maybe()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		// Run with very short intervals
		processor.run(ctx, 10*time.Millisecond, 10*time.Millisecond)

		mockService.AssertExpectations(t)
	})
}

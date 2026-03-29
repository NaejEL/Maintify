package rbac

import (
	"context"
	"maintify/core/pkg/logger"
	"time"

	"github.com/google/uuid"
)

// TimeBasedAccessProcessor handles automatic processing of time-based access controls
type TimeBasedAccessProcessor struct {
	rbacService RBACService
	stopChan    chan struct{}
	doneChan    chan struct{}
}

// NewTimeBasedAccessProcessor creates a new time-based access processor
func NewTimeBasedAccessProcessor(rbacService RBACService) *TimeBasedAccessProcessor {
	return &TimeBasedAccessProcessor{
		rbacService: rbacService,
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
	}
}

// Start begins the background processing of time-based access controls
func (p *TimeBasedAccessProcessor) Start() {
	logger.Info("Starting time-based access processor")

	go p.processLoop()
}

// Stop gracefully stops the time-based access processor
func (p *TimeBasedAccessProcessor) Stop() {
	logger.Info("Stopping time-based access processor")
	close(p.stopChan)
	<-p.doneChan
	logger.Info("Time-based access processor stopped")
}

// processLoop is the main processing loop
func (p *TimeBasedAccessProcessor) processLoop() {
	defer close(p.doneChan)

	// Process immediately on startup
	p.processTimeBasedAccess()

	// Set up tickers for regular processing
	activationTicker := time.NewTicker(5 * time.Minute) // Check for activations every 5 minutes
	cleanupTicker := time.NewTicker(1 * time.Hour)      // Cleanup expired assignments every hour

	defer activationTicker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-p.stopChan:
			return

		case <-activationTicker.C:
			p.processScheduledActivations()

		case <-cleanupTicker.C:
			p.cleanupExpiredAssignments()
		}
	}
}

// processTimeBasedAccess performs both activation and cleanup
func (p *TimeBasedAccessProcessor) processTimeBasedAccess() {
	p.processScheduledActivations()
	p.cleanupExpiredAssignments()
}

// processScheduledActivations processes all pending scheduled role activations
func (p *TimeBasedAccessProcessor) processScheduledActivations() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	err := p.rbacService.ProcessScheduledActivations(ctx)
	if err != nil {
		logger.Error("Error processing scheduled activations", err)
		return
	}

	duration := time.Since(start)
	logger.Info("Processed scheduled activations", map[string]interface{}{
		"duration": duration.String(),
	})
}

// cleanupExpiredAssignments cleans up all expired role assignments
func (p *TimeBasedAccessProcessor) cleanupExpiredAssignments() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	err := p.rbacService.CleanupExpiredAssignments(ctx)
	if err != nil {
		logger.Error("Error cleaning up expired assignments", err)
		return
	}

	duration := time.Since(start)
	logger.Info("Cleaned up expired assignments", map[string]interface{}{
		"duration": duration.String(),
	})
}

// GetStatus returns the current status of time-based access for an organization
func (p *TimeBasedAccessProcessor) GetStatus(ctx context.Context, orgID string) (*TimeBasedAccessStatus, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, err
	}

	return p.rbacService.GetTimeBasedAccessStatus(ctx, orgUUID)
}

// ProcessNow forces immediate processing of all time-based access controls
func (p *TimeBasedAccessProcessor) ProcessNow() {
	logger.Info("Force processing time-based access controls")
	p.processTimeBasedAccess()
}

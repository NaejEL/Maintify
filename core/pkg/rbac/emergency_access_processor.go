package rbac

import (
	"context"
	"maintify/core/pkg/logger"
	"time"
)

// EmergencyAccessProcessor handles background processing for emergency access
type EmergencyAccessProcessor struct {
	rbacService RBACService
	stopCh      chan struct{}
	doneCh      chan struct{}
}

// NewEmergencyAccessProcessor creates a new emergency access processor
func NewEmergencyAccessProcessor(rbacService RBACService) *EmergencyAccessProcessor {
	return &EmergencyAccessProcessor{
		rbacService: rbacService,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// Start begins the background processing
func (p *EmergencyAccessProcessor) Start(ctx context.Context) {
	go p.run(ctx, 5*time.Minute, 1*time.Minute)
}

// run executes the processing loop with specified intervals
func (p *EmergencyAccessProcessor) run(ctx context.Context, escalationInterval, expirationInterval time.Duration) {
	logger.Info("Starting Emergency Access Processor")

	// Process escalations
	escalationTicker := time.NewTicker(escalationInterval)
	defer escalationTicker.Stop()

	// Process expiring requests
	expirationTicker := time.NewTicker(expirationInterval)
	defer expirationTicker.Stop()

	for {
		select {
		case <-escalationTicker.C:
			if err := p.processEscalations(ctx); err != nil {
				logger.Error("Error processing emergency access escalations", err)
			}

		case <-expirationTicker.C:
			if err := p.processExpirations(ctx); err != nil {
				logger.Error("Error processing emergency access expirations", err)
			}

		case <-p.stopCh:
			logger.Info("Emergency Access Processor stopping")
			close(p.doneCh)
			return

		case <-ctx.Done():
			logger.Info("Emergency Access Processor context cancelled")
			close(p.doneCh)
			return
		}
	}
}

// Stop stops the background processing
func (p *EmergencyAccessProcessor) Stop() {
	close(p.stopCh)
	<-p.doneCh
}

// processEscalations handles escalation rules for pending emergency access requests
func (p *EmergencyAccessProcessor) processEscalations(ctx context.Context) error {
	logger.Info("Processing emergency access escalations")
	return p.rbacService.ProcessEmergencyAccessEscalations(ctx)
}

// processExpirations handles expiration of emergency access requests and revocations
func (p *EmergencyAccessProcessor) processExpirations(ctx context.Context) error {
	logger.Info("Processing emergency access expirations")

	// This would typically involve:
	// 1. Expiring pending requests that have reached their expiry time
	// 2. Revoking active emergency access that has exceeded max duration
	// 3. Processing automatic revocations based on break-glass config

	// The database functions handle most of this, but we could add additional logic here
	// for notifications, audit logging, etc.

	return nil
}

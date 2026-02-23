package runtime

// ServiceOption configures a ServiceCtx before it starts running.
type ServiceOption func(*ServiceCtx)

// WithWaitingForServer returns a ServiceOption that creates a ready channel for server startup synchronization.
func WithWaitingForServer() ServiceOption {
	return func(c *ServiceCtx) {
		c.serverReady = make(chan struct{}, 1)
	}
}

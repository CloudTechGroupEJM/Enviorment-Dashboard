package handlers

// webhookDispatcher defines webhook-dispatch capabilities needed by handlers.
// NotificationService satisfies this interface and can be injected where events are triggered.
type webhookDispatcher interface {
	DispatchLifecycleAsync(country string, event string)
	DispatchThresholdAsync(country string, field string, measuredValue float64)
}

package hcore

// SubscribeLogs exposes the core's structured in-memory log stream to the
// daemon adapter. The caller must invoke the returned cleanup function.
func SubscribeLogs(buffer int) (<-chan *LogMessage, func()) {
	updates := static.logObserver.Subscribe(buffer)
	return updates, func() { static.logObserver.Unsubscribe(updates) }
}

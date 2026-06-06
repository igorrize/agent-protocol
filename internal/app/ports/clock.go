package ports

// Clock reports the current time. It is injected so tests can use a fake.
type Clock interface {
	Now() int64 // unix milliseconds
}

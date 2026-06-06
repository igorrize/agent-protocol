package ports

// Spawner launches a locked worker harness for a dispatched task. It is
// fire-and-forget: Spawn returns once the process is started.
type Spawner interface {
	Spawn(task Task, workerToken string, allowedTools []string) error
}

package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"agent-protocol/internal/app/adapters/spawn/harness"
	"agent-protocol/internal/app/ports"
)

// Spawner launches locked worker harnesses via os/exec (fire-and-forget).
type Spawner struct {
	harness  harness.Harness
	proxyURL string
	log      ports.Logger
}

// New builds a Spawner for the given harness and proxy URL.
func New(h harness.Harness, proxyURL string, logger ports.Logger) *Spawner {
	return &Spawner{harness: h, proxyURL: proxyURL, log: logger}
}

// Spawn writes a locked MCP config, then starts the harness for the task with
// stdout+stderr redirected to /tmp/ap-worker-<task>.log. Fire-and-forget: it
// returns once the process is started.
func (s *Spawner) Spawn(task ports.Task, workerToken string, allowedTools []string) error {
	cfgPath, err := s.writeLockedConfig(task.ID, workerToken)
	if err != nil {
		return fmt.Errorf("write locked config: %w", err)
	}

	argv := s.harness.Command(cfgPath, workerPrompt(task.ID), allowedTools)
	if len(argv) == 0 {
		return fmt.Errorf("harness produced empty command")
	}

	logPath := fmt.Sprintf("/tmp/ap-worker-%s.log", task.ID)
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("create worker log: %w", err)
	}

	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is built from a fixed harness template
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start worker: %w", err)
	}
	// Reap the child and close the log when it exits, so we leak neither a
	// zombie process nor the file descriptor.
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
	}()

	s.log.Info("spawned worker", "task", task.ID, "tools", allowedTools, "log", logPath)
	return nil
}

// writeLockedConfig writes the worker's MCP config to a temp file: ONLY our
// proxy, authed with the worker token.
func (s *Spawner) writeLockedConfig(taskID, workerToken string) (string, error) {
	cfg := map[string]any{
		"mcpServers": map[string]any{
			"agent-protocol": map[string]any{
				"type":    "http",
				"url":     s.proxyURL,
				"headers": map[string]any{"Authorization": "Bearer " + workerToken},
			},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", fmt.Sprintf("ap-worker-%s-*.json", taskID))
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// workerPrompt tells the worker how to use its tools.
func workerPrompt(taskID string) string {
	return fmt.Sprintf(`You are a worker agent talking to a proxy.
1) Call listen with {"task_id":"%s"} to fetch your assignment (params + prompt).
2) Do what the assignment's prompt asks (use Read/Grep on the given files if provided).
3) Call complete with {"task_id":"%s","output":{...}} matching the contract.`, taskID, taskID)
}

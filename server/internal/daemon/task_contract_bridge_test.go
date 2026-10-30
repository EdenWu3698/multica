package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// A test-binary entry point lets the App API suite run the real daemon's
// unexported runTask without adding any production endpoint or exported seam.
// Credentials exist only in a parent-owned 0600 file, never command arguments.
func w54BridgeProcess() (int, bool) {
	input := os.Getenv("W54_DAEMON_BRIDGE_INPUT")
	if input == "" {
		return 0, false
	}
	err := func() error {
		var in struct {
			URL, Token, Root, Executable, Source, Marker, Output string
			Task                                                 Task
		}
		raw, err := os.ReadFile(input)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &in); err != nil {
			return err
		}
		u, err := url.Parse(in.URL)
		if err != nil {
			return err
		}
		if u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" && u.Hostname() != "::1") {
			return fmt.Errorf("bridge requires loopback App API")
		}
		for _, path := range []string{in.Executable, in.Source, in.Marker, in.Output} {
			relative, err := filepath.Rel(in.Root, path)
			if err != nil || relative == ".." || filepath.IsAbs(relative) || len(relative) > 3 && relative[:3] == ".."+string(os.PathSeparator) {
				return fmt.Errorf("bridge file outside synthetic root")
			}
		}
		before, err := os.ReadFile(in.Source)
		if err != nil {
			return err
		}
		if in.Task.Agent == nil {
			return fmt.Errorf("real claim has no agent")
		}
		in.Task.Agent.CustomEnv = map[string]string{"W54_PROCESS_MARKER": in.Marker, "W54_SOURCE": in.Source}
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		d := New(Config{ServerBaseURL: in.URL, WorkspacesRoot: filepath.Join(in.Root, "workspaces"), AgentTimeout: 5 * time.Second, Agents: map[string]AgentEntry{"claude": {Path: in.Executable}}}, logger)
		d.executionEnvironmentCommand = nil
		d.client.SetToken(in.Token)
		defer d.client.CloseIdleConnections()
		d.runtimeIndex[in.Task.RuntimeID] = Runtime{ID: in.Task.RuntimeID, Provider: "claude"}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, runErr := d.runTask(ctx, in.Task, "claude", 0, logger)
		after, err := os.ReadFile(in.Source)
		if err != nil {
			return err
		}
		marker, err := os.ReadFile(in.Marker)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		beforeHash, afterHash := sha256.Sum256(before), sha256.Sum256(after)
		result := map[string]any{"executions": len(marker), "before_sha256": hex.EncodeToString(beforeHash[:]), "after_sha256": hex.EncodeToString(afterHash[:]), "run_error": ""}
		if runErr != nil {
			result["run_error"] = runErr.Error()
		}
		raw, err = json.Marshal(result)
		if err != nil {
			return err
		}
		return os.WriteFile(in.Output, raw, 0600)
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, "W54 daemon bridge infrastructure:", err)
		return 1, true
	}
	return 0, true
}

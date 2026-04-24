package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// xmageOracle manages the Java CrossValOracle subprocess.
type xmageOracle struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Scanner
	stderr io.ReadCloser
}

func findJava() string {
	// Try common locations
	candidates := []string{
		"/opt/homebrew/opt/openjdk/bin/java",
		"/opt/homebrew/bin/java",
		"/usr/local/opt/openjdk/bin/java",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Fall back to PATH
	if p, err := exec.LookPath("java"); err == nil {
		return p
	}
	return "java"
}

func launchOracle(xmageDir string, verbose bool) (*xmageOracle, error) {
	// Resolve to absolute path so classpath entries work regardless of cwd
	absDir, err := filepath.Abs(xmageDir)
	if err != nil {
		return nil, fmt.Errorf("resolve xmage dir: %w", err)
	}
	xmageDir = absDir

	classpath, err := buildClasspath(xmageDir)
	if err != nil {
		return nil, fmt.Errorf("build classpath: %w", err)
	}

	javaPath := findJava()
	if verbose {
		fmt.Fprintf(os.Stderr, "Using java: %s\n", javaPath)
	}
	cmd := exec.Command(javaPath, "-cp", classpath, "org.mage.test.crossval.CrossValOracle")
	cmd.Dir = filepath.Join(xmageDir, "Mage.Tests")

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start java: %w", err)
	}

	// Forward stderr in background
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			if verbose {
				fmt.Fprintf(os.Stderr, "[xmage] %s\n", scanner.Text())
			}
		}
	}()

	reader := bufio.NewScanner(stdoutPipe)
	reader.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	return &xmageOracle{
		cmd:    cmd,
		stdin:  stdinPipe,
		reader: reader,
		stderr: stderrPipe,
	}, nil
}

func (o *xmageOracle) send(msg any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	_, err = fmt.Fprintf(o.stdin, "%s\n", b)
	return err
}

func (o *xmageOracle) recv(timeout time.Duration) (*oracleMsg, error) {
	done := make(chan struct{})
	var scanOK bool

	go func() {
		scanOK = o.reader.Scan()
		close(done)
	}()

	select {
	case <-done:
		if !scanOK {
			if err := o.reader.Err(); err != nil {
				return nil, fmt.Errorf("read: %w", err)
			}
			return nil, fmt.Errorf("oracle process closed stdout")
		}
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for oracle response (%v)", timeout)
	}

	line := o.reader.Text()
	var msg oracleMsg
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil, fmt.Errorf("unmarshal oracle response: %w (line: %s)", err, truncate(line, 200))
	}
	return &msg, nil
}

func (o *xmageOracle) close() {
	o.stdin.Close()
	// Give the process a moment to exit gracefully, then force-kill
	done := make(chan struct{})
	go func() {
		o.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		o.cmd.Process.Kill()
		<-done
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func buildClasspath(xmageDir string) (string, error) {
	// Get maven dependency classpath
	mvnCmd := exec.Command("mvn", "-q", "dependency:build-classpath",
		"-pl", "Mage.Tests",
		"-DincludeScope=test",
		"-Dmdep.outputFile=/dev/stdout")
	mvnCmd.Dir = xmageDir

	out, err := mvnCmd.Output()
	if err != nil {
		return "", fmt.Errorf("mvn dependency:build-classpath: %w", err)
	}

	// Parse the output - maven may print extra lines, the classpath is the last non-empty line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	depClasspath := ""
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.HasPrefix(line, "[") {
			depClasspath = line
			break
		}
	}

	// Add module target/classes and target/test-classes directories
	modules := []struct{ path, subdir string }{
		{"Mage", "target/classes"},
		{"Mage.Common", "target/classes"},
		{"Mage.Sets", "target/classes"},
		{"Mage.Server", "target/classes"},
		{"Mage.Server.Plugins/Mage.Player.AI", "target/classes"},
		{"Mage.Server.Plugins/Mage.Game.TwoPlayerDuel", "target/classes"},
		{"Mage.Tests", "target/test-classes"},
	}

	parts := []string{}
	if depClasspath != "" {
		parts = append(parts, depClasspath)
	}
	for _, m := range modules {
		parts = append(parts, filepath.Join(xmageDir, m.path, m.subdir))
	}

	return strings.Join(parts, ":"), nil
}

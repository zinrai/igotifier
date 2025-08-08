package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	version       = "0.1.0"
	debounceDelay = 100 * time.Millisecond
)

type config struct {
	path    string
	execCmd string
	verbose bool
}

func main() {
	cfg := parseFlags()

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() *config {
	var (
		path    = flag.String("path", "", "Path to watch (file or directory)")
		execCmd = flag.String("exec", "", "Command to execute on file change")
		verbose = flag.Bool("verbose", false, "Enable verbose logging")
		ver     = flag.Bool("version", false, "Show version")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "igotifier - File watcher that executes commands on change\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -path=<path> -exec=<command>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s -path=\"/config/app.yaml\" -exec=\"app-reloader sighup --name=nginx\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -path=\"./src\" -exec=\"make test\" -verbose\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *ver {
		fmt.Printf("igotifier version %s\n", version)
		os.Exit(0)
	}

	if *path == "" || *execCmd == "" {
		flag.Usage()
		os.Exit(1)
	}

	return &config{
		path:    *path,
		execCmd: *execCmd,
		verbose: *verbose,
	}
}

func run(cfg *config) error {
	// Validate path exists
	info, err := os.Stat(cfg.path)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", cfg.path, err)
	}

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Add path to watcher
	if info.IsDir() {
		if err := addDir(watcher, cfg.path); err != nil {
			return fmt.Errorf("failed to watch directory: %w", err)
		}
	} else {
		if err := watcher.Add(cfg.path); err != nil {
			return fmt.Errorf("failed to watch file: %w", err)
		}
	}

	log.Printf("Watching %q for changes...", cfg.path)
	if cfg.verbose {
		log.Printf("Will execute: %s", cfg.execCmd)
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Event processing with debouncing
	var (
		timer *time.Timer
		mu    sync.Mutex
	)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Note: We're NOT filtering chmod events to support 'touch' command
			// This may increase event frequency but ensures touch detection

			if cfg.verbose {
				log.Printf("Event: %s on %s", event.Op, event.Name)
			}

			// Debounce: reset timer on each event
			mu.Lock()
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceDelay, func() {
				executeCommand(cfg.execCmd, cfg.verbose)
			})
			mu.Unlock()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)

		case sig := <-sigCh:
			log.Printf("Received signal %v, shutting down...", sig)
			return nil
		}
	}
}

func addDir(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && path != dir {
			return filepath.SkipDir
		}

		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch %q: %w", path, err)
			}
		}

		return nil
	})
}

func executeCommand(cmdStr string, verbose bool) {
	log.Printf("Executing: %s", cmdStr)

	// Use shell to handle complex commands with pipes, redirects, etc.
	cmd := exec.Command("sh", "-c", cmdStr)

	// Capture output
	output, err := cmd.CombinedOutput()

	if verbose && len(output) > 0 {
		log.Printf("Command output:\n%s", string(output))
	}

	if err != nil {
		log.Printf("Command failed: %v", err)
		if !verbose && len(output) > 0 {
			log.Printf("Output:\n%s", string(output))
		}
	} else {
		log.Printf("Command executed successfully")
	}
}

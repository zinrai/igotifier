# igotifier

A lightweight file watcher that executes commands on file changes. Built with Go and powered by fsnotify.

## Features

- Watch files or directories for changes
- Execute arbitrary commands when changes are detected
- Automatic debouncing to prevent command spam
- Recursive directory watching
- Verbose mode for debugging
- Clean shutdown on SIGINT/SIGTERM

## Installation

```bash
$ go install github.com/zinrai/igotifier@latest
```

## Usage

Basic usage:

```bash
$ igotifier -path="/path/to/watch" -exec="command to execute"
```

### Examples

Watch a configuration file and reload nginx:

```bash
$ igotifier -path="/etc/nginx/nginx.conf" -exec="nginx -s reload"
```

Watch a directory and run tests:

```bash
$ igotifier -path="./src" -exec="go test ./..." -verbose
```

Watch for config changes and trigger another tool:

```bash
$ igotifier -path="/config/app.yaml" -exec="app-reloader sighup --name=myapp"
```

### Options

- `-path` (required): File or directory path to watch
- `-exec` (required): Command to execute on file change
- `-verbose`: Enable verbose logging
- `-version`: Show version information

## How it works

1. `igotifier` uses fsnotify to monitor file system events
2. When a change is detected, it triggers the specified command
3. Multiple rapid changes are debounced (100ms delay) to prevent command spam
4. Commands are executed through the shell, allowing complex operations

## Use Cases

- **Configuration reload**: Automatically reload services when config files change
- **Development workflow**: Run tests, linters, or build tools on file save
- **Deployment triggers**: Execute deployment scripts when marker files change
- **Log processing**: Trigger log analysis when new log files appear

## Notes

- Hidden directories (starting with `.`) are skipped during recursive watches
- Commands are executed with `sh -c`, allowing pipes and redirects
- The tool will exit cleanly on SIGINT (Ctrl+C) or SIGTERM

## License

This project is licensed under the [MIT License](./LICENSE).

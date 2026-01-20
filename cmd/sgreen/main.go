package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/inoki/sgreen/internal/session"
	"github.com/inoki/sgreen/internal/ui"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "new":
		handleNew()
	case "attach":
		handleAttach()
	case "list":
		handleList()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func handleNew() {
	// Find the "--" separator first to separate flags from command
	args := os.Args[2:]
	sepIdx := -1
	for i, arg := range args {
		if arg == "--" {
			sepIdx = i
			break
		}
	}

	// Parse flags (only the part before "--")
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	id := fs.String("id", "", "Session ID (required)")
	
	if sepIdx >= 0 {
		// Parse only up to "--"
		fs.Parse(args[:sepIdx])
	} else {
		// No "--" found, parse all args
		fs.Parse(args)
	}

	if *id == "" {
		fmt.Fprintf(os.Stderr, "Error: --id is required\n")
		os.Exit(1)
	}

	if sepIdx == -1 {
		fmt.Fprintf(os.Stderr, "Error: -- separator required before command\n")
		fmt.Fprintf(os.Stderr, "Usage: sgreen new --id <id> -- <cmd> [args...]\n")
		os.Exit(1)
	}

	if sepIdx == len(args)-1 {
		fmt.Fprintf(os.Stderr, "Error: command required after --\n")
		fmt.Fprintf(os.Stderr, "Usage: sgreen new --id <id> -- <cmd> [args...]\n")
		os.Exit(1)
	}

	cmdPath := args[sepIdx+1]
	cmdArgs := args[sepIdx+2:]

	sess, err := session.New(*id, cmdPath, cmdArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Session %s created (PID: %d)\n", sess.ID, sess.Pid)
}

func handleAttach() {
	fs := flag.NewFlagSet("attach", flag.ExitOnError)
	id := fs.String("id", "", "Session ID (required)")
	fs.Parse(os.Args[2:])

	if *id == "" {
		fmt.Fprintf(os.Stderr, "Error: --id is required\n")
		os.Exit(1)
	}

	sess, err := session.Load(*id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading session: %v\n", err)
		os.Exit(1)
	}

	// Check if PTY process is available
	if sess.GetPTYProcess() == nil {
		fmt.Fprintf(os.Stderr, "Error: session %s has no active PTY process\n", *id)
		fmt.Fprintf(os.Stderr, "The session may have been created in a different process\n")
		os.Exit(1)
	}

	err = ui.Attach(os.Stdin, os.Stdout, os.Stderr, sess)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error attaching to session: %v\n", err)
		os.Exit(1)
	}
}

func handleList() {
	sessions := session.List()

	if len(sessions) == 0 {
		fmt.Println("No active sessions")
		return
	}

	fmt.Printf("%-20s %-10s %-30s %s\n", "ID", "PID", "COMMAND", "CREATED")
	fmt.Println(strings.Repeat("-", 80))

	for _, sess := range sessions {
		cmd := sess.CmdPath
		if len(sess.CmdArgs) > 0 {
			cmd += " " + strings.Join(sess.CmdArgs, " ")
		}
		if len(cmd) > 28 {
			cmd = cmd[:25] + "..."
		}

		created := sess.CreatedAt.Format("2006-01-02 15:04:05")
		pidStr := fmt.Sprintf("%d", sess.Pid)
		if sess.GetPTYProcess() == nil {
			pidStr = "N/A"
		}

		fmt.Printf("%-20s %-10s %-30s %s\n", sess.ID, pidStr, cmd, created)
	}
}

func printUsage() {
	fmt.Println("sgreen - A simplified screen-like terminal multiplexer")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sgreen new --id <id> -- <cmd> [args...]")
	fmt.Println("    Create a new session with the given ID and command")
	fmt.Println()
	fmt.Println("  sgreen attach --id <id>")
	fmt.Println("    Attach to an existing session")
	fmt.Println("    Press Ctrl+A, d to detach")
	fmt.Println()
	fmt.Println("  sgreen list")
	fmt.Println("    List all active sessions")
	fmt.Println()
	fmt.Println("  sgreen help")
	fmt.Println("    Show this help message")
}


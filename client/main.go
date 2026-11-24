// Command tenantzero is the TenantZero red team training platform CLI.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/client/journal"
	"github.com/0x4D67/tenantzero/client/labs"
	"github.com/0x4D67/tenantzero/client/terminal"
	wsclient "github.com/0x4D67/tenantzero/client/ws"
	"github.com/0x4D67/tenantzero/client/config"
	"golang.org/x/term"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "version":
		fmt.Printf("tenantzero %s\n", version)
		return nil

	case "login":
		if len(args) < 2 {
			return fmt.Errorf("usage: tenantzero login <server-url>")
		}
		return cmdLogin(args[1])

	case "labs":
		if len(args) == 1 {
			return cmdLabsList()
		}
		if len(args) >= 3 && args[1] == "install" {
			return cmdLabsInstall(args[2])
		}
		return fmt.Errorf("usage: tenantzero labs [install <lab-id>]")

	case "connect":
		if len(args) < 2 {
			return fmt.Errorf("usage: tenantzero connect <lab-id>")
		}
		return cmdConnect(args[1])

	case "offline":
		if len(args) < 2 {
			return fmt.Errorf("usage: tenantzero offline <lab-id>")
		}
		return cmdOffline(args[1])

	case "journal":
		if len(args) >= 3 && args[1] == "export" {
			return cmdJournalExport(args[2])
		}
		return fmt.Errorf("usage: tenantzero journal export <lab-id>")

	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printUsage() {
	fmt.Print(`Usage: tenantzero <command> [args]

Commands:
  login <server-url>         Authenticate and store JWT
  labs                       List available labs
  labs install <lab-id>      Install a lab locally
  connect <lab-id>           Connect to server and start session
  offline <lab-id>           Run lab offline from local cache
  journal export <lab-id>    Export latest journal as markdown
  version                    Print version
`)
}

// cmdLogin prompts for credentials, authenticates against the server, and stores the JWT.
func cmdLogin(serverURL string) error {
	fmt.Print("Username: ")
	var username string
	fmt.Scanln(&username)

	fmt.Print("Password: ")
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	password := string(raw)

	// Convert ws(s) → http(s).
	httpURL := serverURL
	httpURL = strings.Replace(httpURL, "wss://", "https://", 1)
	httpURL = strings.Replace(httpURL, "ws://", "http://", 1)

	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	resp, err := http.Post(httpURL+"/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(resp.Body, &result); err != nil {
		return fmt.Errorf("login: decode response: %w", err)
	}
	if result.Token == "" {
		return fmt.Errorf("login: server returned empty token")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.ServerURL = serverURL
	cfg.Token = result.Token
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Println("[+] Authenticated. Token stored.")
	return nil
}

func cmdLabsList() error {
	cfg, err := loadAuthedConfig()
	if err != nil {
		return err
	}
	b := labs.NewBrowser(cfg.ServerURL, cfg.Token, cfg.LabsDir)
	return b.List()
}

func cmdLabsInstall(labID string) error {
	cfg, err := loadAuthedConfig()
	if err != nil {
		return err
	}
	b := labs.NewBrowser(cfg.ServerURL, cfg.Token, cfg.LabsDir)
	return b.Install(labID)
}

// cmdConnect connects to the server, opens the engagement PDF, and starts the session loop.
func cmdConnect(labID string) error {
	cfg, err := loadAuthedConfig()
	if err != nil {
		return err
	}

	b := labs.NewBrowser(cfg.ServerURL, cfg.Token, cfg.LabsDir)
	if err := b.OpenPDF(labID); err != nil {
		// Non-fatal: PDF may not be installed yet.
		fmt.Fprintf(os.Stderr, "[!] Could not open engagement PDF: %v\n", err)
	}

	c := wsclient.New(cfg.ServerURL, cfg.Token, labID)
	if err := c.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer c.Close()

	renderer := terminal.NewRenderer("WORKSTATION", "", `C:\`)
	renderer.ClearScreen()
	fmt.Printf("[*] Connected to %s — lab: %s\n\n", cfg.ServerURL, labID)

	j, err := journal.New(cfg.JournalDir, labID)
	if err != nil {
		return fmt.Errorf("journal: %w", err)
	}
	defer j.Close()

	var history []string
	prompt := renderer.Prompt()

	for {
		input, err := terminal.ReadLine(prompt, history)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, terminal.ErrInterrupt) {
				continue
			}
			return err
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		history = append(history, input)

		if input == "exit" || input == "quit" {
			break
		}
		if input == "clear" || input == "cls" {
			renderer.ClearScreen()
			continue
		}

		if err := c.Send(input); err != nil {
			return fmt.Errorf("send: %w", err)
		}

		msgType, eventType, payload, err := c.Recv()
		if err != nil {
			return fmt.Errorf("recv: %w", err)
		}

		switch msgType {
		case "response":
			renderer.Print(payload)
			_ = j.Append(journal.Entry{
				Input:     input,
				Output:    payload,
				Timestamp: time.Now(),
			})
			prompt = renderer.Prompt()

		case "event":
			renderer.PrintEvent(eventType, payload)
			if eventType == "lab_complete" {
				return nil
			}
		}
	}
	return nil
}

func cmdOffline(labID string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	r := labs.NewOfflineRunner(labID, cfg.LabsDir)
	return r.Run()
}

func cmdJournalExport(labID string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	j, err := journal.New(cfg.JournalDir, labID)
	if err != nil {
		return err
	}
	defer j.Close()
	return j.ExportMarkdown(os.Stdout)
}

func loadAuthedConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("not authenticated — run 'tenantzero login <server-url>' first")
	}
	return cfg, nil
}

// decodeJSON decodes JSON from r into v.
func decodeJSON(r io.Reader, v any) error {
	buf, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}
// Package labs provides lab browsing, installation, and offline execution.
package labs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

// labMeta is the wire shape returned by GET /labs.
type labMeta struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Difficulty string   `json:"difficulty"`
	Machines   int      `json:"machines"`
	Concepts   []string `json:"concepts"`
	TimeLimit  int      `json:"time_limit_minutes"`
}

// Browser lists and installs labs from the TenantZero server.
type Browser struct {
	serverURL string
	token     string
	cacheDir  string
	client    *http.Client
}

// NewBrowser constructs a Browser.
func NewBrowser(serverURL, token, cacheDir string) *Browser {
	// Convert ws(s) → http(s) for REST calls.
	httpURL := serverURL
	httpURL = strings.Replace(httpURL, "wss://", "https://", 1)
	httpURL = strings.Replace(httpURL, "ws://", "http://", 1)
	return &Browser{
		serverURL: httpURL,
		token:     token,
		cacheDir:  cacheDir,
		client:    &http.Client{},
	}
}

func (b *Browser) get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, b.serverURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	return b.client.Do(req)
}

// List fetches lab metadata from GET /labs and prints a formatted table.
func (b *Browser) List() error {
	resp, err := b.get("/labs")
	if err != nil {
		return fmt.Errorf("browser: list labs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("browser: server returned %d", resp.StatusCode)
	}

	var labs []labMeta
	if err := json.NewDecoder(resp.Body).Decode(&labs); err != nil {
		return fmt.Errorf("browser: decode response: %w", err)
	}

	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tDIFFICULTY\tMACHINES\tTIME (min)\tCONCEPTS")
	fmt.Fprintln(tw, "----\t----------\t--------\t----------\t--------")
	for _, l := range labs {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%s\n",
			l.Name,
			l.Difficulty,
			l.Machines,
			l.TimeLimit,
			strings.Join(l.Concepts, ", "),
		)
	}
	return tw.Flush()
}

// Install downloads a lab's engagement PDF into cacheDir/<labID>/.
func (b *Browser) Install(labID string) error {
	dest := filepath.Join(b.cacheDir, labID)
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return fmt.Errorf("browser: mkdir %s: %w", dest, err)
	}

	// Fetch lab metadata to confirm the lab exists.
	resp, err := b.get("/labs/" + labID)
	if err != nil {
		return fmt.Errorf("browser: fetch lab %s: %w", labID, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("browser: lab %s not found (HTTP %d)", labID, resp.StatusCode)
	}

	// Download engagement PDF.
	pdfResp, err := b.get("/labs/" + labID + "/engagement.pdf")
	if err != nil {
		return fmt.Errorf("browser: fetch PDF for %s: %w", labID, err)
	}
	defer pdfResp.Body.Close()
	if pdfResp.StatusCode != http.StatusOK {
		return fmt.Errorf("browser: PDF not available for %s (HTTP %d)", labID, pdfResp.StatusCode)
	}

	pdfPath := filepath.Join(dest, "engagement.pdf")
	f, err := os.OpenFile(pdfPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("browser: create PDF file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, pdfResp.Body); err != nil {
		return fmt.Errorf("browser: write PDF: %w", err)
	}

	fmt.Printf("[+] Installed %s to %s\n", labID, dest)
	return nil
}

// OpenPDF opens the engagement PDF for a lab using the OS default PDF viewer.
func (b *Browser) OpenPDF(labID string) error {
	pdfPath := filepath.Join(b.cacheDir, labID, "engagement.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return fmt.Errorf("browser: PDF not installed for %s — run 'tenantzero labs install %s' first", labID, labID)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", pdfPath)
	case "darwin":
		cmd = exec.Command("open", pdfPath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", pdfPath)
	default:
		return fmt.Errorf("browser: unsupported OS %s", runtime.GOOS)
	}
	return cmd.Start()
}
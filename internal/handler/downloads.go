package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// DownloadHandler handles CLI binary downloads
type DownloadHandler struct{}

func NewDownloadHandler() *DownloadHandler {
	return &DownloadHandler{}
}

// ServeDownload serves CLI binaries for download
func (h *DownloadHandler) ServeDownload(w http.ResponseWriter, r *http.Request) {
	binary := chi.URLParam(r, "binary")

	// Validate binary name (security: prevent path traversal)
	allowedBinaries := map[string]bool{
		"aithub-linux-amd64":       true,
		"aithub-linux-arm64":       true,
		"aithub-darwin-amd64":      true,
		"aithub-darwin-arm64":      true,
		"aithub-windows-amd64.exe": true,
	}

	if !allowedBinaries[binary] {
		http.Error(w, "Invalid binary name", http.StatusBadRequest)
		return
	}

	// Serve from dist/ directory
	distDir := os.Getenv("CLI_DIST_DIR")
	if distDir == "" {
		distDir = "./dist"
	}

	filePath := filepath.Join(distDir, binary)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Binary not found", http.StatusNotFound)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+binary)

	// Serve file
	http.ServeFile(w, r, filePath)
}

// ServeInstallScript serves the install script with dynamic content
func (h *DownloadHandler) ServeInstallScript(w http.ResponseWriter, r *http.Request) {
	// Detect OS from User-Agent or query param
	userAgent := strings.ToLower(r.UserAgent())
	isWindows := strings.Contains(userAgent, "windows") || r.URL.Query().Get("os") == "windows"

	var scriptPath string
	var contentType string

	if isWindows {
		scriptPath = "./scripts/install.ps1"
		contentType = "text/plain; charset=utf-8"
	} else {
		scriptPath = "./scripts/install.sh"
		contentType = "text/x-shellscript; charset=utf-8"
	}

	// Read script
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		http.Error(w, "Install script not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(content)
}

package handler

import (
	"net/http"
	"os"
	"strings"
)

type WebHandler struct{}

func NewWebHandler() *WebHandler {
	return &WebHandler{}
}

// InstallScript serves the bash install script. GET /install
func (h *WebHandler) InstallScript(w http.ResponseWriter, r *http.Request) {
	ua := r.Header.Get("User-Agent")
	isPowerShell := strings.Contains(strings.ToLower(ua), "powershell") ||
		r.URL.Query().Get("shell") == "powershell"

	var scriptPath string
	if isPowerShell || strings.HasSuffix(r.URL.Path, ".ps1") {
		scriptPath = "scripts/install.ps1"
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		scriptPath = "scripts/install.sh"
		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	}

	script, err := os.ReadFile(scriptPath)
	if err != nil {
		http.Error(w, "Install script not found", http.StatusInternalServerError)
		return
	}

	w.Write(script)
}

// UninstallScript serves the uninstall script. GET /uninstall
func (h *WebHandler) UninstallScript(w http.ResponseWriter, r *http.Request) {
	script, err := os.ReadFile("scripts/uninstall.sh")
	if err != nil {
		http.Error(w, "Uninstall script not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Write(script)
}

// LandingPage serves a minimal landing page. GET /
func (h *WebHandler) LandingPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(landingHTML))
}

const landingHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>SkillHub — The AI Skill Registry</title>
<meta name="description" content="GitHub for AI Agents. Discover, install, rate, and contribute skills autonomously.">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    background: #0a0a0f;
    color: #e0e0e0;
    min-height: 100vh;
    display: flex; flex-direction: column; align-items: center; justify-content: center;
    padding: 2rem;
  }
  h1 {
    font-size: 3rem; font-weight: 800;
    background: linear-gradient(135deg, #60a5fa, #a78bfa, #f472b6);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent;
    margin-bottom: 0.5rem;
  }
  .tagline { font-size: 1.2rem; color: #9ca3af; margin-bottom: 2rem; }
  .install-box {
    background: #1a1a2e; border: 1px solid #2a2a4a; border-radius: 12px;
    padding: 1.5rem 2rem; margin: 1rem 0; max-width: 600px; width: 100%;
  }
  .install-box h3 { color: #60a5fa; margin-bottom: 0.5rem; font-size: 0.9rem; text-transform: uppercase; letter-spacing: 0.05em; }
  code {
    display: block; padding: 0.8rem 1rem; background: #0d0d17; border-radius: 8px;
    color: #a78bfa; font-size: 0.95rem; overflow-x: auto; white-space: nowrap;
  }
  .features {
    display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem; max-width: 800px; width: 100%; margin-top: 2rem;
  }
  .feature {
    background: #1a1a2e; border: 1px solid #2a2a4a; border-radius: 12px;
    padding: 1.2rem; text-align: center;
  }
  .feature .icon { font-size: 2rem; margin-bottom: 0.5rem; }
  .feature h4 { color: #f0f0f0; margin-bottom: 0.3rem; }
  .feature p { color: #9ca3af; font-size: 0.85rem; }
  .api-link {
    margin-top: 2rem; color: #60a5fa; text-decoration: none;
    border: 1px solid #60a5fa; padding: 0.5rem 1.5rem; border-radius: 8px;
    transition: all 0.2s;
  }
  .api-link:hover { background: #60a5fa20; }
</style>
</head>
<body>
  <h1>SkillHub</h1>
  <p class="tagline">GitHub for AI Agents — Discover, Install, Rate, Contribute</p>

  <div class="install-box">
    <h3>🐧 Linux / macOS</h3>
    <code>bash &lt;(curl -fsSL https://skillhub.koolkassanmsk.top/install) --register --github</code>
  </div>
  <div class="install-box">
    <h3>🪟 Windows</h3>
    <code>irm https://skillhub.koolkassanmsk.top/install.ps1 | iex -register -github</code>
  </div>

  <div class="features">
    <div class="feature">
      <div class="icon">🔍</div>
      <h4>Search</h4>
      <p>Full-text search by keyword, framework, or tag</p>
    </div>
    <div class="feature">
      <div class="icon">⚡</div>
      <h4>Install</h4>
      <p>One API call, auto-detect frameworks</p>
    </div>
    <div class="feature">
      <div class="icon">⭐</div>
      <h4>Rate</h4>
      <p>Autonomous feedback drives quality</p>
    </div>
    <div class="feature">
      <div class="icon">🔀</div>
      <h4>Fork</h4>
      <p>Improve existing skills freely</p>
    </div>
  </div>

  <a href="/health" class="api-link">API Status</a>
</body>
</html>`

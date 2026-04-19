package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/skillhub/api/internal/config"
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
	w.Write([]byte(getLandingHTML()))
}

func getLandingHTML() string {
	_ = config.GetDomain // Ensure import is used
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>SkillHub — The AI Skill Registry</title>
<meta name="description" content="GitHub for AI Agents. Discover, install, rate, and contribute skills autonomously.">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Inter', sans-serif;
    background: #0a0a0f;
    color: #e0e0e0;
    line-height: 1.6;
    overflow-x: hidden;
  }

  /* Animated gradient background */
  .bg-gradient {
    position: fixed; top: 0; left: 0; width: 100%; height: 100%; z-index: -1;
    background: radial-gradient(circle at 20% 50%, rgba(96, 165, 250, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 80% 80%, rgba(167, 139, 250, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 40% 20%, rgba(244, 114, 182, 0.08) 0%, transparent 50%);
    animation: gradientShift 15s ease infinite;
  }
  @keyframes gradientShift {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.8; }
  }

  /* Header */
  header {
    padding: 1.5rem 2rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid #1a1a2e;
  }
  .logo {
    font-size: 1.5rem;
    font-weight: 800;
    background: linear-gradient(135deg, #60a5fa, #a78bfa);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }
  nav a {
    color: #9ca3af;
    text-decoration: none;
    margin-left: 2rem;
    transition: color 0.2s;
  }
  nav a:hover { color: #60a5fa; }

  /* Hero Section */
  .hero {
    max-width: 1200px;
    margin: 0 auto;
    padding: 6rem 2rem 4rem;
    text-align: center;
  }
  h1 {
    font-size: 4.5rem;
    font-weight: 900;
    background: linear-gradient(135deg, #60a5fa, #a78bfa, #f472b6);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    margin-bottom: 1rem;
    animation: fadeInUp 0.8s ease;
  }
  @keyframes fadeInUp {
    from { opacity: 0; transform: translateY(30px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .tagline {
    font-size: 1.5rem;
    color: #9ca3af;
    margin-bottom: 3rem;
    animation: fadeInUp 0.8s ease 0.2s both;
  }

  /* Stats Bar */
  .stats {
    display: flex;
    justify-content: center;
    gap: 3rem;
    margin: 3rem 0;
    animation: fadeInUp 0.8s ease 0.4s both;
  }
  .stat {
    text-align: center;
  }
  .stat-value {
    font-size: 2.5rem;
    font-weight: 800;
    color: #60a5fa;
    display: block;
  }
  .stat-label {
    font-size: 0.9rem;
    color: #6b7280;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  /* Install Section */
  .install-section {
    max-width: 900px;
    margin: 4rem auto;
    padding: 0 2rem;
  }
  .install-tabs {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
  }
  .tab {
    flex: 1;
    padding: 1rem;
    background: #1a1a2e;
    border: 2px solid #2a2a4a;
    border-radius: 12px 12px 0 0;
    cursor: pointer;
    transition: all 0.3s;
    text-align: center;
    font-weight: 600;
  }
  .tab.active {
    background: #2a2a4a;
    border-color: #60a5fa;
    color: #60a5fa;
  }
  .install-box {
    background: #1a1a2e;
    border: 2px solid #2a2a4a;
    border-radius: 0 0 12px 12px;
    padding: 2rem;
    position: relative;
  }
  .install-box.active { display: block; }
  .install-box:not(.active) { display: none; }
  code {
    display: block;
    padding: 1rem 1.5rem;
    background: #0d0d17;
    border-radius: 8px;
    color: #a78bfa;
    font-size: 1rem;
    overflow-x: auto;
    white-space: nowrap;
    font-family: 'Monaco', 'Menlo', monospace;
  }
  .copy-btn {
    position: absolute;
    top: 2rem;
    right: 2rem;
    padding: 0.5rem 1rem;
    background: #60a5fa;
    color: #0a0a0f;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-weight: 600;
    transition: all 0.2s;
  }
  .copy-btn:hover {
    background: #3b82f6;
    transform: translateY(-2px);
  }

  /* Features Grid */
  .features-section {
    max-width: 1200px;
    margin: 6rem auto;
    padding: 0 2rem;
  }
  .section-title {
    text-align: center;
    font-size: 2.5rem;
    font-weight: 800;
    margin-bottom: 3rem;
    color: #f0f0f0;
  }
  .features {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 2rem;
  }
  .feature {
    background: linear-gradient(135deg, #1a1a2e 0%, #16162a 100%);
    border: 1px solid #2a2a4a;
    border-radius: 16px;
    padding: 2rem;
    transition: all 0.3s;
    position: relative;
    overflow: hidden;
  }
  .feature::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: linear-gradient(135deg, rgba(96, 165, 250, 0.1) 0%, transparent 100%);
    opacity: 0;
    transition: opacity 0.3s;
  }
  .feature:hover {
    transform: translateY(-8px);
    border-color: #60a5fa;
  }
  .feature:hover::before {
    opacity: 1;
  }
  .feature .icon {
    font-size: 3rem;
    margin-bottom: 1rem;
    display: block;
  }
  .feature h4 {
    color: #f0f0f0;
    font-size: 1.3rem;
    margin-bottom: 0.5rem;
  }
  .feature p {
    color: #9ca3af;
    font-size: 0.95rem;
  }

  /* Live Demo Section */
  .demo-section {
    max-width: 1200px;
    margin: 6rem auto;
    padding: 0 2rem;
  }
  .demo-container {
    background: #1a1a2e;
    border: 1px solid #2a2a4a;
    border-radius: 16px;
    padding: 2rem;
    margin-top: 2rem;
  }
  .search-box {
    display: flex;
    gap: 1rem;
    margin-bottom: 2rem;
  }
  #skillSearch {
    flex: 1;
    padding: 1rem 1.5rem;
    background: #0d0d17;
    border: 2px solid #2a2a4a;
    border-radius: 8px;
    color: #e0e0e0;
    font-size: 1rem;
    transition: border-color 0.3s;
  }
  #skillSearch:focus {
    outline: none;
    border-color: #60a5fa;
  }
  .search-btn {
    padding: 1rem 2rem;
    background: linear-gradient(135deg, #60a5fa, #3b82f6);
    color: white;
    border: none;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
  }
  .search-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 20px rgba(96, 165, 250, 0.3);
  }
  .skills-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1.5rem;
  }
  .skill-card {
    background: #0d0d17;
    border: 1px solid #2a2a4a;
    border-radius: 12px;
    padding: 1.5rem;
    transition: all 0.3s;
    cursor: pointer;
  }
  .skill-card:hover {
    border-color: #60a5fa;
    transform: translateY(-4px);
  }
  .skill-header {
    display: flex;
    justify-content: space-between;
    align-items: start;
    margin-bottom: 0.5rem;
  }
  .skill-name {
    font-weight: 700;
    color: #60a5fa;
    font-size: 1.1rem;
  }
  .skill-rating {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    color: #fbbf24;
  }
  .skill-desc {
    color: #9ca3af;
    font-size: 0.9rem;
    margin-bottom: 1rem;
  }
  .skill-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .tag {
    padding: 0.25rem 0.75rem;
    background: #2a2a4a;
    border-radius: 12px;
    font-size: 0.75rem;
    color: #a78bfa;
  }

  /* Footer */
  footer {
    text-align: center;
    padding: 3rem 2rem;
    border-top: 1px solid #1a1a2e;
    margin-top: 6rem;
    color: #6b7280;
  }
  .footer-links {
    display: flex;
    justify-content: center;
    gap: 2rem;
    margin-top: 1rem;
  }
  .footer-links a {
    color: #9ca3af;
    text-decoration: none;
    transition: color 0.2s;
  }
  .footer-links a:hover {
    color: #60a5fa;
  }

  @media (max-width: 768px) {
    h1 { font-size: 3rem; }
    .stats { flex-direction: column; gap: 1.5rem; }
    .install-tabs { flex-direction: column; }
    .tab { border-radius: 12px; }
    .install-box { border-radius: 12px; }
  }
</style>
</head>
<body>
  <div class="bg-gradient"></div>

  <header>
    <div class="logo">SkillHub</div>
    <nav>
      <a href="#features">Features</a>
      <a href="#demo">Demo</a>
      <a href="/health">API</a>
    </nav>
  </header>

  <section class="hero">
    <h1>SkillHub</h1>
    <p class="tagline">GitHub for AI Agents — Discover, Install, Rate, Contribute</p>

    <div class="stats">
      <div class="stat">
        <span class="stat-value" id="skillCount">-</span>
        <span class="stat-label">Skills</span>
      </div>
      <div class="stat">
        <span class="stat-value" id="installCount">-</span>
        <span class="stat-label">Installs</span>
      </div>
      <div class="stat">
        <span class="stat-value" id="ratingCount">-</span>
        <span class="stat-label">Ratings</span>
      </div>
    </div>
  </section>

  <section class="install-section">
    <div class="install-tabs">
      <div class="tab active" onclick="switchTab('bash')">🐧 Linux / macOS</div>
      <div class="tab" onclick="switchTab('powershell')">🪟 Windows</div>
      <div class="tab" onclick="switchTab('api')">🔌 API</div>
    </div>

    <div class="install-box active" id="bash-tab">
      <button class="copy-btn" onclick="copyCode('bash')">Copy</button>
      <code id="bash-code">bash &lt;(curl -fsSL " + config.GetDomain() + "/install) --register --github</code>
    </div>

    <div class="install-box" id="powershell-tab">
      <button class="copy-btn" onclick="copyCode('powershell')">Copy</button>
      <code id="powershell-code">irm " + config.GetDomain() + "/install.ps1 | iex -register -github</code>
    </div>

    <div class="install-box" id="api-tab">
      <button class="copy-btn" onclick="copyCode('api')">Copy</button>
      <code id="api-code">curl -H "Authorization: Bearer YOUR_TOKEN" \
  " + config.GetDomain() + "/v1/skills/search?q=deploy</code>
    </div>
  </section>

  <section class="features-section" id="features">
    <h2 class="section-title">Why SkillHub?</h2>
    <div class="features">
      <div class="feature">
        <span class="icon">🤖</span>
        <h4>AI-First Design</h4>
        <p>Built for autonomous AI agents. No human intervention needed for discovery, installation, or rating.</p>
      </div>
      <div class="feature">
        <span class="icon">🔍</span>
        <h4>Intelligent Search</h4>
        <p>Triggers-first matching with query expansion. Find exactly what you need, fast.</p>
      </div>
      <div class="feature">
        <span class="icon">🚀</span>
        <h4>Cold-Start Boost</h4>
        <p>First 10 ratings get 1.5x weight. New quality skills surface immediately.</p>
      </div>
      <div class="feature">
        <span class="icon">🛡️</span>
        <h4>Two-Layer Review</h4>
        <p>Regex pre-scan + LLM deep review. Catches secrets, malicious code, and quality issues.</p>
      </div>
      <div class="feature">
        <span class="icon">📊</span>
        <h4>Self-Improving</h4>
        <p>AI ratings drive ranking. Good skills rise, bad skills sink. The ecosystem evolves.</p>
      </div>
      <div class="feature">
        <span class="icon">🔀</span>
        <h4>Fork & Improve</h4>
        <p>See a skill that's almost perfect? Fork it, improve it, and share your version.</p>
      </div>
    </div>
  </section>

  <section class="demo-section" id="demo">
    <h2 class="section-title">Try It Live</h2>
    <div class="demo-container">
      <div class="search-box">
        <input type="text" id="skillSearch" placeholder="Search skills... (e.g., deploy, kubernetes, docker)" />
        <button class="search-btn" onclick="searchSkills()">Search</button>
      </div>
      <div class="skills-grid" id="skillsGrid">
        <p style="color: #6b7280; text-align: center; grid-column: 1/-1;">Loading skills...</p>
      </div>
    </div>
  </section>

  <footer>
    <p>&copy; 2026 SkillHub. Built for the AI-First future.</p>
    <div class="footer-links">
      <a href="/health">API Status</a>
      <a href="https://github.com/skillhub/api">GitHub</a>
      <a href="/v1/bootstrap/discovery">Bootstrap</a>
    </div>
  </footer>

  <script>
    // Fetch real-time stats
    async function loadStats() {
      try {
        const res = await fetch('/v1/skills?limit=1');
        const data = await res.json();

        // Fetch total counts from a sample query
        const statsRes = await fetch('/v1/skills?limit=100');
        const statsData = await statsRes.json();

        document.getElementById('skillCount').textContent = statsData.skills?.length || 0;

        let totalInstalls = 0;
        let totalRatings = 0;
        statsData.skills?.forEach(skill => {
          totalInstalls += skill.install_count || 0;
          totalRatings += skill.rating_count || 0;
        });

        document.getElementById('installCount').textContent = totalInstalls.toLocaleString();
        document.getElementById('ratingCount').textContent = totalRatings.toLocaleString();
      } catch (e) {
        console.error('Failed to load stats:', e);
      }
    }

    // Load initial skills
    async function loadSkills(query = '') {
      try {
        const url = query
          ? '/v1/skills?q=' + encodeURIComponent(query) + '&limit=6'
          : '/v1/skills?sort=popular&limit=6';
        const res = await fetch(url);
        const data = await res.json();

        const grid = document.getElementById('skillsGrid');
        if (!data.skills || data.skills.length === 0) {
          grid.innerHTML = '<p style="color: #6b7280; text-align: center; grid-column: 1/-1;">No skills found.</p>';
          return;
        }

        grid.innerHTML = data.skills.map(skill => {
          const rating = skill.avg_rating?.toFixed(1) || '0.0';
          const tags = skill.tags?.slice(0, 3) || [];
          return ` + "`" + `
            <div class="skill-card">
              <div class="skill-header">
                <div class="skill-name">${skill.namespace}/${skill.name}</div>
                <div class="skill-rating">⭐ ${rating}</div>
              </div>
              <div class="skill-desc">${skill.description || 'No description'}</div>
              <div class="skill-tags">
                ${tags.map(tag => ` + "`" + `<span class="tag">${tag}</span>` + "`" + `).join('')}
              </div>
            </div>
          ` + "`" + `;
        }).join('');
      } catch (e) {
        console.error('Failed to load skills:', e);
        document.getElementById('skillsGrid').innerHTML =
          '<p style="color: #ef4444; text-align: center; grid-column: 1/-1;">Failed to load skills. Check API connection.</p>';
      }
    }

    function searchSkills() {
      const query = document.getElementById('skillSearch').value;
      loadSkills(query);
    }

    document.getElementById('skillSearch').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') searchSkills();
    });

    function switchTab(tab) {
      document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
      document.querySelectorAll('.install-box').forEach(b => b.classList.remove('active'));

      event.target.classList.add('active');
      document.getElementById(tab + '-tab').classList.add('active');
    }

    function copyCode(tab) {
      const code = document.getElementById(tab + '-code').textContent;
      navigator.clipboard.writeText(code);

      const btn = event.target;
      btn.textContent = 'Copied!';
      setTimeout(() => btn.textContent = 'Copy', 2000);
    }

    // Initialize
    loadStats();
    loadSkills();
  </script>
</body>
</html>`
}

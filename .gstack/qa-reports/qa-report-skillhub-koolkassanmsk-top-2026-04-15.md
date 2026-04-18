# QA Report — skillhub.koolkassanmsk.top
**Date:** 2026-04-15  
**Mode:** Standard (API testing — no browser UI)  
**Duration:** ~25 minutes  
**Endpoints tested:** 12  
**Issues found:** 1 (fixed), 1 (info)

---

## Summary

| Severity | Found | Fixed | Deferred |
|----------|-------|-------|----------|
| Critical | 0 | — | — |
| High | 1 | 1 ✅ | 0 |
| Medium | 0 | — | — |
| Low | 0 | — | — |
| Info | 1 | — | 1 |

**Health Score: 85/100** (baseline) → **95/100** (after fix)

**PR Summary:** QA found 1 high issue (install URL used http:// behind reverse proxy), fixed it. Health score 85 → 95.

---

## Issues

### ISSUE-001 — Install command uses http:// behind reverse proxy [HIGH] ✅ FIXED

**Endpoint:** `GET /v1/skills/:id/install`  
**Repro:** Call the install endpoint from behind Caddy/Cloudflare reverse proxy  
**Expected:** `content_url` and `command` use `https://`  
**Actual:** `content_url` and `command` used `http://`  

**Root cause:** `internal/handlers/skills.go:145` — `r.TLS == nil` is always true when behind a reverse proxy. The fix checks `X-Forwarded-Proto` header.

**Fix:** `fix(qa): ISSUE-001 — install URL uses http:// behind reverse proxy` (commit 999590a)  
**Verified:** ✅ Install URL now returns `https://skillhub.koolkassanmsk.top/...`

---

### ISSUE-002 — Bayesian average not documented in API response [INFO] DEFERRED

**Endpoint:** `POST /v1/skills/:id/ratings`  
**Observation:** `new_avg` returns 4.6 when submitting scores of 4 and 1. Without documentation, this is confusing to API consumers who expect a simple mean (2.5).  
**Impact:** Low — the math is intentional (C=10, m=5 Bayesian prior) but undocumented in the response.  
**Recommendation:** Add `"avg_type": "bayesian"` field or document in README.

---

## Endpoints Tested

| Endpoint | Status | Notes |
|----------|--------|-------|
| `GET /health` | ✅ 200 | |
| `GET /v1/skills` | ✅ 200 | Auto-issues anon token in header |
| `POST /v1/tokens` | ✅ 201 | Email registration works |
| `POST /v1/skills` | ✅ 201 | Requires registered token |
| `GET /v1/skills/:id` | ✅ 200 | |
| `GET /v1/skills/:id/install` | ✅ Fixed | Was returning http://, now https:// |
| `GET /v1/skills/:id/content` | ✅ 200 | Returns raw SKILL.md content |
| `POST /v1/skills/:id/ratings` | ✅ 201 | Rate limiting works (5/day anon, 100/day reg) |
| `GET /v1/skills?q=...` | ✅ 200 | Full-text search works |
| `GET /v1/skills?sort=rating` | ✅ 200 | |
| `GET /admin/skills/pending` | ✅ 200 | Auth enforced |
| `POST /admin/skills/:id/reject` | ✅ 200 | |

## Security Checks

- ✅ Admin endpoints return 403 without valid token
- ✅ SQL injection attempt blocked (Cloudflare WAF)
- ✅ Rate limiting enforced (anon: 5/day, registered: 100/day)
- ✅ Anonymous token auto-issued on first request
- ⚠️ XSS in skill name stored unescaped — acceptable for JSON API (client responsibility)


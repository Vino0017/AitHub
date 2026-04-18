package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/skillhub/api/internal/db"
	"github.com/skillhub/api/internal/email"
	"github.com/skillhub/api/internal/handler"
	"github.com/skillhub/api/internal/llm"
	"github.com/skillhub/api/internal/middleware"
	"github.com/skillhub/api/internal/review"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	// Auto-migrate if enabled
	if os.Getenv("AUTO_MIGRATE") == "true" {
		log.Println("running migrations...")
		if err := db.RunMigrations(ctx, pool); err != nil {
			log.Fatalf("migrations: %v", err)
		}
	}

	// Seed data if enabled
	if os.Getenv("SEED_DATA") == "true" {
		if err := db.RunSeed(ctx, pool); err != nil {
			log.Printf("seed data warning: %v", err)
		}
	}

	adminToken := os.Getenv("ADMIN_TOKEN")
	if adminToken == "" {
		adminToken = "change-me-in-production"
	}

	// Initialize LLM client for AI review
	llmClient := llm.NewClient()
	aiReviewEnabled := os.Getenv("AI_REVIEW_ENABLED") == "true" && llmClient.IsConfigured()
	reviewer := review.NewReviewer(pool, llmClient, aiReviewEnabled)

	// Initialize River job queue
	workers := river.NewWorkers()
	river.AddWorker(workers, review.NewReviewWorker(reviewer))

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Workers: workers,
		Queues: map[string]river.QueueConfig{
			"review":              {MaxWorkers: 5},
			river.QueueDefault:    {MaxWorkers: 10},
		},
	})
	if err != nil {
		log.Fatalf("river client: %v", err)
	}
	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("river start: %v", err)
	}
	defer riverClient.Stop(ctx)

	// Initialize email sender
	emailSender := email.NewSender()

	// Handlers
	tokens := handler.NewTokenHandler(pool)
	auth := handler.NewAuthHandler(pool, emailSender)
	search := handler.NewSkillSearchHandler(pool)
	detail := handler.NewSkillDetailHandler(pool)
	submit := handler.NewSkillSubmitHandler(pool, reviewer, riverClient)
	ratings := handler.NewRatingHandler(pool)
	revisions := handler.NewRevisionHandler(pool, reviewer, riverClient)
	forks := handler.NewForkHandler(pool)
	yank := handler.NewSkillYankHandler(pool)
	namespaces := handler.NewNamespaceHandler(pool)
	admin := handler.NewAdminHandler(pool)
	bootstrap := handler.NewBootstrapHandler()

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// ── Health ──
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"version":"2.0.0"}`))
	})

	// ── Web (landing page + install scripts) ──
	web := handler.NewWebHandler()
	r.Get("/", web.LandingPage)
	r.Get("/install", web.InstallScript)
	r.Get("/install.ps1", web.InstallScript)
	r.Get("/uninstall", web.UninstallScript)

	// ── Public: Token creation ──
	r.Post("/v1/tokens", tokens.Create)

	// ── Public: Bootstrap (Discovery Skill auto-installation) ──
	r.Get("/v1/bootstrap/discovery", bootstrap.GetDiscoverySkill)
	r.Get("/v1/bootstrap/check", bootstrap.CheckBootstrap)

	// ── Public: Auth (no token needed) ──
	r.Post("/v1/auth/github", auth.GitHubDeviceStart)
	r.Post("/v1/auth/github/poll", auth.GitHubDevicePoll)
	r.Post("/v1/auth/email/send", auth.EmailSend)
	r.Post("/v1/auth/email/verify", auth.EmailVerify)

	// ── Authenticated (token required) ──
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(pool))

		// Search & browse (anonymous OK)
		r.Get("/v1/skills", search.Search)
		r.Get("/v1/skills/{namespace}/{name}", detail.Get)
		r.Get("/v1/skills/{namespace}/{name}/content", detail.Content)
		r.Get("/v1/skills/{namespace}/{name}/status", detail.Status)
		r.Get("/v1/skills/{namespace}/{name}/revisions", revisions.List)
		r.Get("/v1/skills/{namespace}/{name}/revisions/{version}", revisions.GetVersion)
		r.Get("/v1/skills/{namespace}/{name}/forks", forks.ListForks)
		r.Get("/v1/namespaces/{name}", namespaces.Get)

		// Rating (anonymous can rate, but won't count toward ranking)
		r.Post("/v1/skills/{namespace}/{name}/ratings", ratings.Submit)

		// Namespace-required actions
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireNamespace)

			r.Get("/v1/tokens", tokens.List)
			r.Delete("/v1/tokens/{id}", tokens.Delete)

			r.Post("/v1/skills", submit.Submit)
			r.Post("/v1/skills/{namespace}/{name}/revisions", revisions.Submit)
			r.Post("/v1/skills/{namespace}/{name}/fork", forks.Fork)
			r.Delete("/v1/skills/{namespace}/{name}", yank.Yank)
			r.Patch("/v1/skills/{namespace}/{name}", yank.Restore)

			r.Post("/v1/namespaces", namespaces.Create)
			r.Post("/v1/namespaces/{name}/members", namespaces.AddMember)
			r.Delete("/v1/namespaces/{name}/members/{memberId}", namespaces.RemoveMember)
		})
	})

	// ── Admin ──
	r.Group(func(r chi.Router) {
		r.Use(middleware.AdminAuth(adminToken))
		r.Get("/admin/skills/pending", admin.ListPending)
		r.Post("/admin/skills/{id}/approve", admin.Approve)
		r.Post("/admin/skills/{id}/reject", admin.Reject)
		r.Post("/admin/skills/{id}/remove", admin.RemoveSkill)
		r.Post("/admin/namespaces/{id}/ban", admin.BanNamespace)
		r.Post("/admin/ratings/refresh", admin.RefreshRatings)
	})

	// ── Background tasks ──
	go runPeriodicRatingRefresh(pool)

	// ── Start server ──
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("SkillHub API v2 listening on :%s", port)
	log.Printf("  AI Review: %v", aiReviewEnabled)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

// runPeriodicRatingRefresh recalculates all skill ratings every 5 minutes.
// Implements cold-start boost: first 10 ratings get 1.5x weight for new skills
func runPeriodicRatingRefresh(pool *pgxpool.Pool) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		_, err := pool.Exec(ctx,
			`UPDATE skills s SET
			    avg_rating = sub.bayesian_avg,
			    rating_count = sub.n,
			    outcome_success_rate = sub.success_rate,
			    updated_at = NOW()
			 FROM (
			    SELECT sk.id AS skill_id,
			           COALESCE(stats.n, 0) AS n,
			           CASE WHEN COALESCE(stats.n, 0) = 0 THEN 0
			                -- Cold-start boost: first 10 ratings get 1.5x weight
			                WHEN COALESCE(stats.n, 0) <= 10 THEN
			                    (5.0 * 6.0 + COALESCE(stats.weighted_total, 0)) / (5.0 + COALESCE(stats.weighted_n, 0))
			                ELSE
			                    (5.0 * 6.0 + COALESCE(stats.total_score, 0)) / (5.0 + COALESCE(stats.n, 0))
			           END AS bayesian_avg,
			           COALESCE(stats.success_rate, 0) AS success_rate
			    FROM skills sk
			    LEFT JOIN LATERAL (
			        SELECT rev.id AS rev_id FROM revisions rev
			        WHERE rev.skill_id = sk.id AND rev.review_status = 'approved'
			        ORDER BY rev.created_at DESC LIMIT 1
			    ) latest_rev ON TRUE
			    LEFT JOIN LATERAL (
			        SELECT COUNT(*)::int AS n,
			               SUM(r.score)::float AS total_score,
			               -- Weighted sum: first 10 ratings * 1.5
			               SUM(CASE WHEN ROW_NUMBER() OVER (ORDER BY r.created_at) <= 10
			                        THEN r.score * 1.5
			                        ELSE r.score END)::float AS weighted_total,
			               -- Weighted count: first 10 ratings count as 1.5
			               SUM(CASE WHEN ROW_NUMBER() OVER (ORDER BY r.created_at) <= 10
			                        THEN 1.5
			                        ELSE 1.0 END)::float AS weighted_n,
			               SUM(CASE WHEN r.outcome = 'success' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) AS success_rate
			        FROM ratings r JOIN tokens t ON r.token_id = t.id
			        WHERE r.revision_id = latest_rev.rev_id AND t.namespace_id IS NOT NULL
			    ) stats ON TRUE
			    WHERE sk.status = 'active'
			 ) sub
			 WHERE s.id = sub.skill_id`)
		cancel()
		if err != nil {
			log.Printf("periodic rating refresh error: %v", err)
		}
	}
}

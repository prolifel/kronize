package server

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"kronize/internal/auth"
	"kronize/internal/db"
	"kronize/internal/handler"
	"kronize/internal/logstream"
	"kronize/internal/scheduler"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	DB         *sql.DB
	Addr       string
	JWTSecret  string
	ScriptsDir string
	Scheduler  *scheduler.Scheduler
	StreamHub  *logstream.Hub
}

func New(database *sql.DB, addr, jwtSecret, scriptsDir string) *Server {
	hub := logstream.New(database)
	return &Server{
		DB:         database,
		Addr:       addr,
		JWTSecret:  jwtSecret,
		ScriptsDir: scriptsDir,
		Scheduler:  scheduler.New(database, scriptsDir, hub),
		StreamHub:  hub,
	}
}

func (s *Server) Start() error {
	enabledJobs, err := db.ListEnabledJobs(s.DB)
	if err != nil {
		return err
	}
	for _, j := range enabledJobs {
		s.Scheduler.AddJob(j)
	}
	go s.Scheduler.Start()

	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Use(auth.Middleware(s.JWTSecret), handler.RequirePasswordChanged(s.DB))

		r.Post("/auth/login", handler.Login(s.DB, s.JWTSecret))
		r.Post("/auth/logout", handler.Logout())
		r.Get("/auth/me", handler.GetCurrentUser(s.DB))
		r.Post("/auth/change-password", handler.ChangePassword(s.DB))

		r.Get("/jobs", handler.ListJobs(s.DB))
		r.Post("/jobs", handler.CreateJob(s.DB, s.Scheduler, s.ScriptsDir))
		r.Get("/jobs/{id}", handler.GetJob(s.DB))
		r.Put("/jobs/{id}", handler.UpdateJob(s.DB, s.Scheduler, s.ScriptsDir))
		r.Delete("/jobs/{id}", handler.DeleteJob(s.DB, s.Scheduler))
		r.Post("/jobs/{id}/run", handler.RunJob(s.DB, s.Scheduler))
		r.Put("/jobs/{id}/toggle", handler.ToggleJob(s.DB, s.Scheduler))

		r.Get("/jobs/{id}/executions", handler.ListExecutions(s.DB))
		r.Get("/executions/{id}", handler.GetExecution(s.DB))
		r.Get("/executions/{id}/stream", handler.StreamExecution(s.DB, s.StreamHub))

		r.Get("/stats", handler.GetStats(s.DB))

		r.Get("/settings", handler.GetSettings(s.DB))
		r.Put("/settings", handler.UpdateSettings(s.DB))

		r.Get("/runners", handler.ListRunnerImages(s.DB))

		r.Group(func(r chi.Router) {
			r.Use(auth.AdminOnly)
			r.Get("/users", handler.ListUsers(s.DB))
			r.Post("/users", handler.CreateUser(s.DB))
			r.Put("/users/{id}", handler.UpdateUser(s.DB))
			r.Delete("/users/{id}", handler.DeleteUser(s.DB))
			r.Post("/runners", handler.CreateRunnerImage(s.DB))
			r.Put("/runners/{id}", handler.UpdateRunnerImage(s.DB))
			r.Delete("/runners/{id}", handler.DeleteRunnerImage(s.DB))
		})
	})

	dist := "./frontend/dist"
	if _, err := os.Stat(dist); err == nil {
		fileServer := http.FileServer(http.Dir(dist))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(dist, r.URL.Path)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				r.URL.Path = "/"
			}
			if !strings.Contains(r.URL.Path, ".") {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
	} else {
		slog.Info("frontend dist not found, API only mode")
	}

	slog.Info("listening", "addr", s.Addr)
	return http.ListenAndServe(s.Addr, r)
}

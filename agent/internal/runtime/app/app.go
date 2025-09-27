package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	pluginspec "github.com/ccheshirecat/volant/pkg/pluginspec"
	"github.com/volant-plugins/browser/internal/runtime/browser"
)

const (
	defaultListenAddr     = ":8080"
	defaultTimeoutEnvKey  = "volant_AGENT_DEFAULT_TIMEOUT"
	defaultListenEnvKey   = "volant_AGENT_LISTEN_ADDR"
	defaultRemoteAddrKey  = "volant_AGENT_REMOTE_DEBUGGING_ADDR"
	defaultRemotePortKey  = "volant_AGENT_REMOTE_DEBUGGING_PORT"
	defaultUserDataDirKey = "volant_AGENT_USER_DATA_DIR"
	defaultExecPathKey    = "volant_AGENT_EXEC_PATH"
)

type Config struct {
	ListenAddr          string
	RemoteDebuggingAddr string
	RemoteDebuggingPort int
	UserDataDir         string
	ExecPath            string
	DefaultTimeout      time.Duration
}

type App struct {
	cfg     Config
	runtime *browser.Runtime
	timeout time.Duration
	log     *log.Logger
	started time.Time
}

func Run(ctx context.Context) error {
	cfg := loadConfig()
	logger := log.New(os.Stdout, "browser-runtime: ", log.LstdFlags|log.LUTC)

	manifest, err := resolveManifest()
	if err != nil {
		return err
	}

	options := browser.Options{
		DefaultTimeout: cfg.DefaultTimeout,
		RemoteAddr:     cfg.RemoteDebuggingAddr,
		RemotePort:     cfg.RemoteDebuggingPort,
		UserDataDir:    cfg.UserDataDir,
		ExecPath:       cfg.ExecPath,
	}
	if manifest != nil {
		options.Manifest = manifest
	}

	runtimeInstance, err := browser.New(ctx, options)
	if err != nil {
		return err
	}
	defer runtimeInstance.Shutdown(context.Background())

	app := &App{
		cfg:     cfg,
		runtime: runtimeInstance,
		timeout: cfg.DefaultTimeout,
		log:     logger,
		started: time.Now().UTC(),
	}
	return app.run(ctx)
}

func (a *App) run(ctx context.Context) error {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(a.timeout + 30*time.Second))

	router.Get("/healthz", a.handleHealth)

	router.Route("/v1", func(r chi.Router) {
		r.Get("/devtools", a.handleDevTools)
		r.Get("/logs/stream", a.handleLogs)
		a.runtime.MountRoutes(r)
	})

	server := &http.Server{
		Addr:         a.cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if info, ok := a.runtime.DevToolsInfo(); ok {
			a.log.Printf("listening on %s (devtools ws: %s)", a.cfg.ListenAddr, info.WebSocketURL)
		} else {
			a.log.Printf("listening on %s", a.cfg.ListenAddr)
		}
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			a.log.Printf("shutdown error: %v", err)
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func loadConfig() Config {
	remoteAddr := envOrDefault(defaultRemoteAddrKey, browser.DefaultRemoteAddr)
	remotePort := envIntOrDefault(defaultRemotePortKey, browser.DefaultRemotePort)
	defaultTimeout := parseDurationEnv(defaultTimeoutEnvKey, browser.DefaultActionTimeout)

	return Config{
		ListenAddr:          envOrDefault(defaultListenEnvKey, defaultListenAddr),
		RemoteDebuggingAddr: remoteAddr,
		RemoteDebuggingPort: remotePort,
		UserDataDir:         os.Getenv(defaultUserDataDirKey),
		ExecPath:            os.Getenv(defaultExecPathKey),
		DefaultTimeout:      defaultTimeout,
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
		if ms, err := strconv.ParseInt(value, 10, 64); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return fallback
}

func resolveManifest() (*pluginspec.Manifest, error) {
	encoded := strings.TrimSpace(os.Getenv("volant_MANIFEST"))
	if encoded == "" {
		return nil, nil
	}
	manifest, err := pluginspec.Decode(encoded)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"uptime":  time.Since(a.started).Round(time.Second).String(),
		"version": "v0.1.0",
	})
}

func (a *App) handleDevTools(w http.ResponseWriter, r *http.Request) {
	info, ok := a.runtime.DevToolsInfo()
	if !ok {
		errorJSON(w, http.StatusNotFound, errors.New("runtime does not expose devtools"))
		return
	}
	respondJSON(w, http.StatusOK, info)
}

func (a *App) handleLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		errorJSON(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}

	ch, unsubscribe := a.runtime.SubscribeLogs(128)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				a.log.Printf("log stream marshal error: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func errorJSON(w http.ResponseWriter, status int, err error) {
	respondJSON(w, status, map[string]any{"error": err.Error()})
}

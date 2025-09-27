package browser

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	pluginspec "github.com/ccheshirecat/volant/pkg/pluginspec"
)

const Name = "browser"

// Options configure the browser runtime.
type Options struct {
	DefaultTimeout time.Duration
	RemoteAddr     string
	RemotePort     int
	UserDataDir    string
	ExecPath       string
	Manifest       *pluginspec.Manifest
}

// Runtime exposes HTTP handlers backed by the Browser automation engine.
type Runtime struct {
	real           *Browser
	defaultTimeout time.Duration
	manifest       *pluginspec.Manifest
}

// New constructs a new browser runtime.
func New(ctx context.Context, opts Options) (*Runtime, error) {
	cfg := BrowserConfig{}
	if opts.RemoteAddr != "" {
		cfg.RemoteDebuggingAddr = opts.RemoteAddr
	}
	if opts.RemotePort != 0 {
		cfg.RemoteDebuggingPort = opts.RemotePort
	}
	cfg.UserDataDir = opts.UserDataDir
	cfg.ExecPath = opts.ExecPath
	cfg.DefaultTimeout = opts.DefaultTimeout

	browser, err := NewBrowser(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Runtime{
		real:           browser,
		defaultTimeout: cfg.DefaultTimeout,
		manifest:       opts.Manifest,
	}, nil
}

// Name returns the runtime identifier.
func (r *Runtime) Name() string { return Name }

// DevToolsInfo exposes DevTools metadata.
func (r *Runtime) DevToolsInfo() (DevToolsInfo, bool) { return r.real.DevToolsInfo(), true }

// SubscribeLogs relays runtime logs.
func (r *Runtime) SubscribeLogs(buffer int) (<-chan LogEvent, func()) {
	return r.real.SubscribeLogs(buffer)
}

// BrowserInstance returns the underlying browser controller.
func (r *Runtime) BrowserInstance() *Browser { return r.real }

// MountRoutes registers HTTP handlers.
func (r *Runtime) MountRoutes(router chi.Router) {
	r.mountRoutes(router, r.manifest)
}

func (r *Runtime) MountRoutesWithManifest(router chi.Router, manifest pluginspec.Manifest) error {
	r.manifest = &manifest
	r.mountRoutes(router, r.manifest)
	return nil
}

func (r *Runtime) mountRoutes(router chi.Router, manifest *pluginspec.Manifest) {
	router.Route("/browser", r.mountBrowserRoutes)
	router.Route("/dom", r.mountDOMRoutes)
	router.Route("/script", r.mountScriptRoutes)
	router.Route("/profile", r.mountProfileRoutes)
}

// Shutdown terminates the runtime.
func (r *Runtime) Shutdown(ctx context.Context) error {
	r.real.Close()
	return nil
}

func (r *Runtime) duration(ms int64) time.Duration {
	if ms <= 0 {
		if r.defaultTimeout > 0 {
			return r.defaultTimeout
		}
		return DefaultActionTimeout
	}
	return time.Duration(ms) * time.Millisecond
}

func parseInt(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("browser runtime: invalid int %q: %w", value, err)
	}
	return parsed, nil
}

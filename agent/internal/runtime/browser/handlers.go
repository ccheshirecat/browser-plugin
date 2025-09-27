package browser

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"	
        "time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/go-chi/chi/v5"
)

	func (r *Runtime) mountBrowserRoutes(router chi.Router) {
	router.Post("/navigate", func(w http.ResponseWriter, req *http.Request) {
		var payload navigateRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(payload.URL) == "" {
			errorJSON(w, http.StatusBadRequest, errors.New("url is required"))
			return
		}
		if err := r.real.Navigate(r.duration(payload.TimeoutMs), payload.URL); err != nil {
			errorJSON(w, http.StatusInternalServerError, err)
			return
		}
		okJSON(w)
	})

	router.Handle(http.MethodPost, "/reload", func(w http.ResponseWriter, req *http.Request) {
		var payload reloadRequest
		_ = decodeRequest(req, &payload)
		if err := r.real.Reload(r.duration(payload.TimeoutMs), payload.IgnoreCache); err != nil {
			errorJSON(w, http.StatusInternalServerError, err)
			return
		}
		okJSON(w)
	})

	router.Handle(http.MethodPost, "/back", func(w http.ResponseWriter, req *http.Request) {
		if err := r.real.Back(r.duration(queryTimeout(req))); err != nil {
			errorJSON(w, http.StatusInternalServerError, err)
			return
		}
		okJSON(w)
	})

	router.Handle(http.MethodPost, "/forward", func(w http.ResponseWriter, req *http.Request) {
		if err := r.real.Forward(r.duration(queryTimeout(req))); err != nil {
			errorJSON(w, http.StatusInternalServerError, err)
			return
		}
		okJSON(w)
	})

	router.Handle(http.MethodPost, "/viewport", func(w http.ResponseWriter, req *http.Request) {
		var payload viewportRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if err := r.real.SetViewport(r.duration(payload.TimeoutMs), payload.Width, payload.Height, payload.Scale, payload.Mobile); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		okJSON(w)
	})

	router.Handle(http.MethodPost, "/user-agent", func(w http.ResponseWriter, req *http.Request) {
		var payload userAgentRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if err := r.real.SetUserAgent(r.duration(payload.TimeoutMs), payload.UserAgent, payload.AcceptLanguage, payload.Platform); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		okJSON(w)
	})

	router.Handle(http.MethodPost, "/wait-navigation", func(w http.ResponseWriter, req *http.Request) {
		var payload waitNavigationRequest
		_ = decodeRequest(req, &payload)
		if err := r.real.WaitForNavigation(r.duration(payload.TimeoutMs)); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		okJSON(w)
	})

	router.Post("/screenshot", func(w http.ResponseWriter, req *http.Request) {
		var payload screenshotRequest
		_ = decodeRequest(req, &payload)
		data, err := r.real.Screenshot(r.duration(payload.TimeoutMs), payload.FullPage, payload.Format, payload.Quality)
		if err != nil {
			errorJSON(w, http.StatusInternalServerError, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"data":        base64.StdEncoding.EncodeToString(data),
			"format":      strings.ToLower(payload.Format),
			"full_page":   payload.FullPage,
			"byte_length": len(data),
			"captured_at": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})
}

func (r *Runtime) mountDOMRoutes(router chi.Router) {
	router.Post("/click", func(w http.ResponseWriter, req *http.Request) {
		var payload clickRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if err := r.real.Click(r.duration(payload.TimeoutMs), payload.Selector, payload.Button); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		okJSON(w)
	})

	router.Post("/type", func(w http.ResponseWriter, req *http.Request) {
		var payload typeRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if err := r.real.Type(r.duration(payload.TimeoutMs), payload.Selector, payload.Value, payload.Clear); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		okJSON(w)
	})

	router.Post("/get-text", func(w http.ResponseWriter, req *http.Request) {
		var payload textRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		text, err := r.real.GetText(r.duration(payload.TimeoutMs), payload.Selector, payload.Visible)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"text": text})
	})

	router.Post("/get-html", func(w http.ResponseWriter, req *http.Request) {
		var payload htmlRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		html, err := r.real.GetHTML(r.duration(payload.TimeoutMs), payload.Selector)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"html": html})
	})

	router.Post("/get-attribute", func(w http.ResponseWriter, req *http.Request) {
		var payload attributeRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		value, ok, err := r.real.GetAttribute(r.duration(payload.TimeoutMs), payload.Selector, payload.Name)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": value, "exists": ok})
	})

	router.Post("/wait-selector", func(w http.ResponseWriter, req *http.Request) {
		var payload waitSelectorRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if err := r.real.WaitForSelector(r.duration(payload.TimeoutMs), payload.Selector, payload.Visible); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		okJSON(w)
	})
}

func (r *Runtime) mountScriptRoutes(router chi.Router) {
	router.Post("/evaluate", func(w http.ResponseWriter, req *http.Request) {
		var payload evaluateRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(payload.Expression) == "" {
			errorJSON(w, http.StatusBadRequest, errors.New("expression is required"))
			return
		}
		result, err := r.real.Evaluate(r.duration(payload.TimeoutMs), payload.Expression, payload.AwaitPromise)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"result": result})
	})
}

func (r *Runtime) mountActionRoutes(router chi.Router) {
	router.Post("/navigate", func(w http.ResponseWriter, req *http.Request) {
		var payload navigateRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(payload.URL) == "" {
			errorJSON(w, http.StatusBadRequest, errors.New("url is required"))
			return
		}
		if err := r.real.Navigate(r.duration(payload.TimeoutMs), payload.URL); err != nil {
			errorJSON(w, http.StatusInternalServerError, err)
			return
		}
		okJSON(w)
	})

	router.Post("/screenshot", func(w http.ResponseWriter, req *http.Request) {
		var payload screenshotRequest
		_ = decodeRequest(req, &payload)
		data, err := r.real.Screenshot(r.duration(payload.TimeoutMs), payload.FullPage, payload.Format, payload.Quality)
		if err != nil {
			errorJSON(w, http.StatusInternalServerError, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"data":        base64.StdEncoding.EncodeToString(data),
			"format":      strings.ToLower(payload.Format),
			"full_page":   payload.FullPage,
			"byte_length": len(data),
			"captured_at": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	router.Post("/scrape", func(w http.ResponseWriter, req *http.Request) {
		var payload scrapeRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(payload.Selector) == "" {
			errorJSON(w, http.StatusBadRequest, errors.New("selector is required"))
			return
		}
		timeout := r.duration(payload.TimeoutMs)
		if attr := strings.TrimSpace(payload.Attribute); attr != "" {
			value, exists, err := r.real.GetAttribute(timeout, payload.Selector, attr)
			if err != nil {
				errorJSON(w, http.StatusBadRequest, err)
				return
			}
			respondJSON(w, http.StatusOK, map[string]any{"value": value, "exists": exists})
			return
		}
		text, err := r.real.GetText(timeout, payload.Selector, true)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": text, "exists": true})
	})

	router.Post("/evaluate", func(w http.ResponseWriter, req *http.Request) {
		var payload evaluateRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(payload.Expression) == "" {
			errorJSON(w, http.StatusBadRequest, errors.New("expression is required"))
			return
		}
		result, err := r.real.Evaluate(r.duration(payload.TimeoutMs), payload.Expression, payload.AwaitPromise)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"result": result})
	})

	router.Post("/graphql", func(w http.ResponseWriter, req *http.Request) {
		var payload graphqlRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(payload.Endpoint) == "" {
			errorJSON(w, http.StatusBadRequest, errors.New("endpoint is required"))
			return
		}
		if strings.TrimSpace(payload.Query) == "" {
			errorJSON(w, http.StatusBadRequest, errors.New("query is required"))
			return
		}
		if payload.Variables == nil {
			payload.Variables = map[string]any{}
		}
		response, err := r.real.GraphQL(r.duration(payload.TimeoutMs), payload.Endpoint, payload.Query, payload.Variables)
		if err != nil {
			errorJSON(w, http.StatusBadGateway, err)
			return
		}
		respondJSON(w, http.StatusOK, response)
	})
}

func (r *Runtime) mountProfileRoutes(router chi.Router) {
	router.Post("/attach", func(w http.ResponseWriter, req *http.Request) {
		var payload profileAttachRequest
		if err := decodeRequest(req, &payload); err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}

		var cookies []*network.CookieParam
		for _, c := range payload.Cookies {
			cookie, err := convertCookieParam(c)
			if err != nil {
				errorJSON(w, http.StatusBadRequest, err)
				return
			}
			cookies = append(cookies, cookie)
		}

		timeout := r.duration(payload.Timeout)
		if len(cookies) > 0 {
			if err := r.real.SetCookies(timeout, cookies); err != nil {
				errorJSON(w, http.StatusBadRequest, err)
				return
			}
		}

		if len(payload.Local) > 0 || len(payload.Session) > 0 {
			storagePayload := StoragePayload{Local: payload.Local, Session: payload.Session}
			if err := r.real.SetStorage(timeout, storagePayload); err != nil {
				errorJSON(w, http.StatusBadRequest, err)
				return
			}
		}

		okJSON(w)
	})

	router.Get("/extract", func(w http.ResponseWriter, req *http.Request) {
		cookies, err := r.real.GetCookies(r.duration(queryTimeout(req)))
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}
		storage, err := r.real.GetStorage(r.defaultTimeout)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err)
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"cookies":         mapCookies(cookies),
			"local_storage":   storage.Local,
			"session_storage": storage.Session,
		})
	})
}

func decodeRequest[T any](r *http.Request, dest *T) error {
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(dest)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func errorJSON(w http.ResponseWriter, status int, err error) {
	respondJSON(w, status, map[string]any{"error": err.Error()})
}

func okJSON(w http.ResponseWriter) {
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func queryTimeout(r *http.Request) int64 {
	if value := r.URL.Query().Get("timeout_ms"); value != "" {
		if ms, err := strconv.ParseInt(value, 10, 64); err == nil && ms > 0 {
			return ms
		}
	}
	return 0
}

// Request payloads ----------------------------------------------------------

type navigateRequest struct {
	URL       string `json:"url"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type reloadRequest struct {
	IgnoreCache bool  `json:"ignore_cache"`
	TimeoutMs   int64 `json:"timeout_ms"`
}

type viewportRequest struct {
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Scale     float64 `json:"scale"`
	Mobile    bool    `json:"mobile"`
	TimeoutMs int64   `json:"timeout_ms"`
}

type userAgentRequest struct {
	UserAgent      string `json:"user_agent"`
	AcceptLanguage string `json:"accept_language"`
	Platform       string `json:"platform"`
	TimeoutMs      int64  `json:"timeout_ms"`
}

type waitNavigationRequest struct {
	TimeoutMs int64 `json:"timeout_ms"`
}

type screenshotRequest struct {
	FullPage  bool   `json:"full_page"`
	Format    string `json:"format"`
	Quality   int    `json:"quality"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type clickRequest struct {
	Selector  string `json:"selector"`
	Button    string `json:"button"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type typeRequest struct {
	Selector  string `json:"selector"`
	Value     string `json:"value"`
	Clear     bool   `json:"clear"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type textRequest struct {
	Selector  string `json:"selector"`
	Visible   bool   `json:"visible"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type htmlRequest struct {
	Selector  string `json:"selector"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type attributeRequest struct {
	Selector  string `json:"selector"`
	Name      string `json:"name"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type waitSelectorRequest struct {
	Selector  string `json:"selector"`
	Visible   bool   `json:"visible"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type evaluateRequest struct {
	Expression   string `json:"expression"`
	AwaitPromise bool   `json:"await_promise"`
	TimeoutMs    int64  `json:"timeout_ms"`
}

type scrapeRequest struct {
	Selector  string `json:"selector"`
	Attribute string `json:"attribute"`
	TimeoutMs int64  `json:"timeout_ms"`
}

type graphqlRequest struct {
	Endpoint  string         `json:"endpoint"`
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
	TimeoutMs int64          `json:"timeout_ms"`
}

type cookieParamRequest struct {
	Name     string   `json:"name"`
	Value    string   `json:"value"`
	Domain   string   `json:"domain"`
	Path     string   `json:"path"`
	Expires  *float64 `json:"expires"`
	HTTPOnly bool     `json:"http_only"`
	Secure   bool     `json:"secure"`
	SameSite string   `json:"same_site"`
}

type profileAttachRequest struct {
	Cookies []cookieParamRequest `json:"cookies"`
	Local   map[string]string    `json:"local_storage"`
	Session map[string]string    `json:"session_storage"`
	Timeout int64                `json:"timeout_ms"`
}

func convertCookieParam(req cookieParamRequest) (*network.CookieParam, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("cookie name required")
	}
	cookie := &network.CookieParam{
		Name:     req.Name,
		Value:    req.Value,
		Domain:   req.Domain,
		Path:     req.Path,
		HTTPOnly: req.HTTPOnly,
		Secure:   req.Secure,
	}
	if req.SameSite != "" {
		switch strings.ToLower(req.SameSite) {
		case "lax":
			cookie.SameSite = network.CookieSameSiteLax
		case "strict":
			cookie.SameSite = network.CookieSameSiteStrict
		case "none":
			cookie.SameSite = network.CookieSameSiteNone
		default:
			return nil, errors.New("invalid same_site")
		}
	}
	if req.Expires != nil {
		epoch := cdp.TimeSinceEpoch(time.Unix(int64(*req.Expires), 0).UTC())
		cookie.Expires = &epoch
	}
	return cookie, nil
}

func mapCookies(cookies []*network.Cookie) []map[string]any {
	result := make([]map[string]any, 0, len(cookies))
	for _, c := range cookies {
		item := map[string]any{
			"name":      c.Name,
			"value":     c.Value,
			"domain":    c.Domain,
			"path":      c.Path,
			"http_only": c.HTTPOnly,
			"secure":    c.Secure,
			"same_site": strings.ToLower(string(c.SameSite)),
		}
		item["expires"] = c.Expires
		result = append(result, item)
	}
	return result
}

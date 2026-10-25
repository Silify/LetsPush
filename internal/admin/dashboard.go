package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Silify/letspush/internal/store"
)

const (
	defaultPage = 50
	maxPage     = 200
	recentCount = 8
	dayWindow   = 24 * time.Hour
)

// logActions are the kinds of entry the log filter accepts.
var logActions = map[string]bool{"app": true, "accesskey": true, "token": true, "push": true, "error": true}

type logView struct {
	ID      int64     `json:"id"`
	At      time.Time `json:"at"`
	Action  string    `json:"action"`
	Info    string    `json:"info"`
	AppID   *int64    `json:"appId"`
	AppName string    `json:"appName"`
}

func logViews(entries []store.LogEntry) []logView {
	views := make([]logView, 0, len(entries))
	for _, e := range entries {
		views = append(views, logView{e.ID, e.CreatedAt, e.Action, e.Info, e.AppID, e.AppName})
	}
	return views
}

// page is a list that may continue: Next is the id to pass as "before" for the following page, or null.
type page[T any] struct {
	Items []T    `json:"items"`
	Next  *int64 `json:"next"`
}

// pageParams reads "limit" and "before" from the query. It answers the error itself.
func pageParams(w http.ResponseWriter, r *http.Request) (limit int, before int64, ok bool) {
	limit = defaultPage
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxPage {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return 0, 0, false
		}
		limit = n
	}
	if raw := r.URL.Query().Get("before"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "before must be an id")
			return 0, 0, false
		}
		before = n
	}
	return limit, before, true
}

// logs is GET /admin/api/logs (every app, or one with ?app=ID) and GET /admin/api/apps/{id}/logs.
// Both take ?limit, ?before and ?action.
func (h *Handler) logs(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	var appID int64
	if r.PathValue("id") != "" {
		id, ok := pathID(r, "id")
		if !ok {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		appID = id
	} else if raw := r.URL.Query().Get("app"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			writeError(w, http.StatusBadRequest, "app must be an id")
			return
		}
		appID = id
	}
	limit, before, ok := pageParams(w, r)
	if !ok {
		return
	}
	action := r.URL.Query().Get("action")
	if action != "" && !logActions[action] {
		writeError(w, http.StatusBadRequest, "unknown action")
		return
	}
	entries, err := h.store.ListLogs(store.LogFilter{AppID: appID, Action: action, BeforeID: before, Limit: limit})
	if err != nil {
		storeError(w, err)
		return
	}
	out := page[logView]{Items: logViews(entries)}
	if len(entries) == limit {
		out.Next = &entries[len(entries)-1].ID
	}
	writeJSON(w, http.StatusOK, out)
}

// overview is GET /admin/api/overview: the numbers and the latest activity for the dashboard's first page.
func (h *Handler) overview(w http.ResponseWriter, _ *http.Request, _ store.Admin) {
	totals, err := h.store.Totals()
	if err != nil {
		storeError(w, err)
		return
	}
	sent, errs, err := h.dayCounts(0)
	if err != nil {
		storeError(w, err)
		return
	}
	recent, err := h.store.ListLogs(store.LogFilter{Limit: recentCount})
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"apps":           totals.Apps,
		"accessKeys":     totals.Keys,
		"devices":        totals.Devices,
		"appsNeedingKey": totals.AppsWithoutKey,
		"sent24h":        sent,
		"errors24h":      errs,
		"recent":         logViews(recent),
	})
}

// dayCounts is how many notifications were sent and how many sends failed in the last day, for one app
// or (with 0) for all.
func (h *Handler) dayCounts(appID int64) (sent, failed int, err error) {
	since := h.now().Add(-dayWindow)
	if sent, err = h.store.CountLogs(appID, "push", "sent to ", since); err != nil {
		return 0, 0, err
	}
	failed, err = h.store.CountLogs(appID, "error", "", since)
	return sent, failed, err
}

type deviceView struct {
	ID        int64     `json:"id"`
	Token     string    `json:"token"` // only the start of it
	Platform  string    `json:"platform"`
	Channel   string    `json:"channel"`
	Key       deviceKey `json:"key"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type deviceKey struct {
	Prefix      string `json:"prefix"`
	Description string `json:"description"`
	Revoked     bool   `json:"revoked"`
}

// maskToken keeps enough of a device token to recognise it, never more than half of it.
func maskToken(token string) string {
	runes := []rune(token)
	keep := min(8, len(runes)/2)
	return string(runes[:keep]) + "…"
}

// devices is GET /admin/api/apps/{id}/devices: the app's devices, newest first, with the key of each.
func (h *Handler) devices(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	limit, before, ok := pageParams(w, r)
	if !ok {
		return
	}
	if _, err := h.store.AppByID(id); err != nil {
		storeError(w, err)
		return
	}
	rows, err := h.store.ListDevices(id, before, limit)
	if err != nil {
		storeError(w, err)
		return
	}
	out := page[deviceView]{Items: make([]deviceView, 0, len(rows))}
	for _, d := range rows {
		out.Items = append(out.Items, deviceView{
			ID: d.ID, Token: maskToken(d.Token), Platform: d.Platform, Channel: d.Channel,
			Key:       deviceKey{d.KeyPrefix, d.KeyDescription, d.KeyRevoked},
			CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		})
	}
	if len(rows) == limit {
		out.Next = &rows[len(rows)-1].ID
	}
	writeJSON(w, http.StatusOK, out)
}

// removeDevice is DELETE /admin/api/apps/{id}/devices/{deviceId}.
func (h *Handler) removeDevice(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	appID, ok := pathID(r, "id")
	deviceID, ok2 := pathID(r, "deviceId")
	if !ok || !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.store.DeleteDevice(appID, deviceID); err != nil {
		storeError(w, err)
		return
	}
	_ = h.store.AddLog(appID, "token", "removed by an administrator")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

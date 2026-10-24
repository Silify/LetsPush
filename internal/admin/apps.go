package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Silify/letspush/internal/fcm"
	"github.com/Silify/letspush/internal/store"
)

type appView struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	ProjectID        string    `json:"projectId"`
	HasCredentials   bool      `json:"hasCredentials"`
	AllowKeyRequests bool      `json:"allowKeyRequests"`
	CreatedAt        time.Time `json:"createdAt"`
	Keys             int       `json:"keys"`
	Tokens           int       `json:"tokens"`
}

func viewOf(a store.App) appView {
	return appView{a.ID, a.Name, a.ProjectID, a.HasCredentials, a.AllowKeyRequests, a.CreatedAt, a.Keys, a.Tokens}
}

func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id, err == nil && id > 0
}

func (h *Handler) listApps(w http.ResponseWriter, _ *http.Request, _ store.Admin) {
	apps, err := h.store.ListApps()
	if err != nil {
		storeError(w, err)
		return
	}
	views := make([]appView, 0, len(apps))
	for _, a := range apps {
		views = append(views, viewOf(a))
	}
	writeJSON(w, http.StatusOK, views)
}

func (h *Handler) createApp(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	var body struct {
		Name           string `json:"name"`
		ServiceAccount string `json:"serviceAccount"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if !appName.MatchString(name) {
		writeError(w, http.StatusBadRequest, "the name may use letters, digits, dots, dashes and underscores, up to 64 characters")
		return
	}
	projectID, sealed, ok := h.sealCredentials(w, body.ServiceAccount, true)
	if !ok {
		return
	}
	app, err := h.store.CreateApp(name, projectID, sealed)
	if err != nil {
		storeError(w, err)
		return
	}
	_ = h.store.AddLog(app.ID, "app", "created")
	writeJSON(w, http.StatusCreated, viewOf(app))
}

func (h *Handler) showApp(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	app, err := h.store.AppByID(id)
	if err != nil {
		storeError(w, err)
		return
	}
	keys, err := h.store.ListKeys(app.ID)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		appView
		AccessKeys []keyView `json:"accessKeys"`
	}{viewOf(app), keyViews(keys)})
}

func (h *Handler) deleteApp(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.store.DeleteApp(id); err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) setCredentials(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		ServiceAccount string `json:"serviceAccount"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	projectID, sealed, ok := h.sealCredentials(w, body.ServiceAccount, false)
	if !ok {
		return
	}
	if err := h.store.UpdateAppCredentials(id, projectID, sealed); err != nil {
		storeError(w, err)
		return
	}
	_ = h.store.AddLog(id, "app", "Firebase key replaced")
	app, err := h.store.AppByID(id)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, viewOf(app))
}

func (h *Handler) setSettings(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		AllowKeyRequests bool `json:"allowKeyRequests"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := h.store.SetAllowKeyRequests(id, body.AllowKeyRequests); err != nil {
		storeError(w, err)
		return
	}
	app, err := h.store.AppByID(id)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, viewOf(app))
}

// sealCredentials checks an uploaded Firebase key and seals it for storage.
// A missing key is fine when [optional] is set; the app then cannot send yet.
func (h *Handler) sealCredentials(w http.ResponseWriter, raw string, optional bool) (projectID, sealed string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" && optional {
		return "", "", true
	}
	creds, err := fcm.ParseCredentials([]byte(raw))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Firebase key: "+err.Error())
		return "", "", false
	}
	sealed, err = h.sealer.Seal([]byte(raw))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong")
		return "", "", false
	}
	return creds.ProjectID, sealed, true
}

type logView struct {
	At     time.Time `json:"at"`
	Action string    `json:"action"`
	Info   string    `json:"info"`
}

func (h *Handler) logs(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 200 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return
		}
		limit = n
	}
	entries, err := h.store.RecentLogs(id, limit)
	if err != nil {
		storeError(w, err)
		return
	}
	views := make([]logView, 0, len(entries))
	for _, e := range entries {
		views = append(views, logView{e.CreatedAt, e.Action, e.Info})
	}
	writeJSON(w, http.StatusOK, views)
}

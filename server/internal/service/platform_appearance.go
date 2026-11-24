package service

import "net/http"

func (a *API) PlatformAppearance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	theme, err := a.DB.PlatformTheme(r.Context())
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"theme": theme})
}

func (a *API) SetPlatformAppearance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if _, ok := a.platform(w, r); !ok {
		return
	}
	var body struct {
		Theme string `json:"theme"`
	}
	if err := readJSONStrict(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.Theme != "classic" && body.Theme != "exhibition" {
		writeErr(w, http.StatusBadRequest, "invalid_theme")
		return
	}
	if err := a.DB.SetPlatformTheme(r.Context(), body.Theme); err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

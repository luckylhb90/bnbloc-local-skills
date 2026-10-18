package localskills

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// maxSettingsBody bounds a settings update.
const maxSettingsBody = 256 << 10

// Handler serves the skills provider API (ProviderContract) on loopback:
//
//	GET  /v1/state                what the component has (State)
//	PUT  /v1/settings             replaces the settings (Settings), answers State
//	POST /v1/pack?name=<skill>    one skill as a tar, its hash in X-Skill-Hash
//
// token, when set, is the bearer every call must carry (the desktop hands it to
// the component at launch and to nothing else).
func (s *Service) Handler(token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/state", func(w http.ResponseWriter, r *http.Request) {
		st, err := s.State()
		if err != nil {
			slog.Error("server error", "what", "read the skills", "error", err.Error())
			writeError(w, "could not read the skills: "+err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, st)
	})
	mux.HandleFunc("PUT /v1/settings", func(w http.ResponseWriter, r *http.Request) {
		var in Settings
		if !decodeJSON(w, r, maxSettingsBody, &in) {
			return
		}
		st, err := s.Apply(in)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, st)
	})
	mux.HandleFunc("POST /v1/pack", func(w http.ResponseWriter, r *http.Request) {
		archive, hash, err := s.Pack(r.URL.Query().Get("name"))
		if err != nil {
			writeError(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/x-tar")
		w.Header().Set("X-Skill-Hash", hash)
		_, _ = w.Write(archive)
	})
	return requireBearer(token, mux)
}

func requireBearer(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			writeError(w, errors.New("unauthorized").Error(), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeJSON answers status with v as JSON (messages keep <, > and &
// readable: a JSON answer is never embedded in HTML).
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// writeError answers status with the error body the desktop reads
// ({"error": msg}).
func writeError(w http.ResponseWriter, msg string, status int) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// decodeJSON reads a JSON body of at most limit bytes into v, answering 413
// or 400 itself when it cannot.
func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	err := json.NewDecoder(r.Body).Decode(v)
	if err == nil {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, "request body too large", http.StatusRequestEntityTooLarge)
		return false
	}
	writeError(w, "invalid json body", http.StatusBadRequest)
	return false
}

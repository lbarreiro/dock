package api

import (
	"encoding/json"
	"net/http"
	"sync"
)

// One mutation at a time: small hosts must not overlap recreate, prune or stop.
type Operations struct{ mu sync.Mutex }

func NewOperations() *Operations { return &Operations{} }
func (o *Operations) Try() bool  { return o.mu.TryLock() }
func (o *Operations) Done()      { o.mu.Unlock() }
func (o *Operations) Guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !o.Try() {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Another Docker operation is in progress; wait for it to finish."})
			return
		}
		defer o.Done()
		next(w, r)
	}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

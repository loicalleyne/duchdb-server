package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/loicalleyne/bodkin"
)

type syncBodkin struct {
	mu           sync.Mutex
	u            *bodkin.Bodkin
	lastAccessed time.Time
}

var schemas sync.Map

func newSyncBodkin() *syncBodkin {
	sb := new(syncBodkin)
	sb.lastAccessed = time.Now()
	return sb
}

func schemasTTL() {
	time.Sleep(10 * time.Minute)
	for {
		var expired []string
		today := time.Now()
		limit := today.Add(-36 * time.Hour)
		schemas.Range(func(k, v interface{}) bool {
			v.(*syncBodkin).mu.Lock()
			if v.(*syncBodkin).lastAccessed.Before(limit) {
				expired = append(expired, k.(string))
			}
			return true
		})
		for _, k := range expired {
			schemas.Delete(k)
		}
		time.Sleep(6 * time.Hour)
	}
}

func (s *chServer) handleSchemaPost(w http.ResponseWriter, r *http.Request) {
	schemaName := r.URL.Query().Get("name")
	if schemaName == "" {
		http.Error(w, "Missing schema name", http.StatusBadRequest)
		return
	}
	if r.Body != nil {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	if sb, ok := schemas.Load(schemaName); !ok {
		u, err := bodkin.NewBodkin(bodyBytes, bodkin.WithInferTimeUnits(), bodkin.WithTypeConversion())
		if err != nil {
			http.Error(w, "Error creating schema", http.StatusBadRequest)
			return
		}
		sb := newSyncBodkin()
		sb.u = u
		schemas.Store(schemaName, sb)
		w.WriteHeader(http.StatusOK)
		return
	} else {
		sb.(*syncBodkin).mu.Lock()
		sb.(*syncBodkin).lastAccessed = time.Now()
		defer sb.(*syncBodkin).mu.Unlock()
		err := sb.(*syncBodkin).u.Unify(bodyBytes)
		if err != nil {
			http.Error(w, "Error unifying schema", http.StatusBadRequest)
			return
		}
	}
}

func (s *chServer) handleSchemaGet(w http.ResponseWriter, r *http.Request) {
	schemaName := r.URL.Query().Get("name")
	if schemaName == "" {
		http.Error(w, "Missing schema name", http.StatusBadRequest)
		return
	}
	if sb, ok := schemas.Load(schemaName); !ok {
		http.Error(w, "Error creating schema", http.StatusNotFound)
		return

	} else {
		sb.(*syncBodkin).mu.Lock()
		sb.(*syncBodkin).lastAccessed = time.Now()
		defer sb.(*syncBodkin).mu.Unlock()
		arrSchema, err := sb.(*syncBodkin).u.ExportSchemaBytes()
		if err != nil {
			http.Error(w, "Error exporting schema", http.StatusInternalServerError)
			return
		}
		w.Header().Add("content-type", "application/octet-stream")
		fmt.Fprintf(w, "%b", arrSchema)
	}
}

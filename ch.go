package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/loicalleyne/chdbsession"
)

// Embedding the HTML file
//
//go:embed play.html
var content embed.FS

var (
	chDBpath string
	duckPath string
	readOnly string
	ro       bool
)

func init() {
	chDBpath = os.Getenv("DATA_PATH")
	if chDBpath == "" {
		chDBpath = ".chdb_data"
	}
	duckPath = os.Getenv("DUCK_PATH")
	if duckPath == "" {
		duckPath = ".duckdb_data/duck.db"
	}
	readOnly = os.Getenv("READ_ONLY")
	if readOnly == "true" {
		ro = true
	}
	initDuckDB(duckPath)
}

type auth struct {
	isTokenAuth bool
	username    string
	password    string
	token       string
}

type chServer struct {
	sess *chdb.Session
	auth auth
}

func newServer(chDBpath string) *chServer {
	s := &chServer{}
	sess, err := chdb.NewSession(chDBpath)
	if err != nil {
		log.Fatalf("Error creating session: %v", err)
	}
	s.sess = sess
	return s
}

func (s *chServer) handleRootPost(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	format := r.URL.Query().Get("default_format")
	// chdb formats list: https://github.com/chdb-io/chdb/blob/8a84df2b25b2cb4ecdbabf9161f06411802d1d7d/programs/bash-completion/completions/clickhouse-bootstrap#L44
	if format == "" {
		format = "JSONCompact"
	}
	database := r.URL.Query().Get("database")
	var body string
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		body = string(bodyBytes)
	}
	if body != "" {
		body = strings.Join(strings.Fields(body), " ")
		if query != "" {
			query += " "
		}
		query += body
	}
	if database != "" {
		query = "USE " + database + "; " + query
	}
	log.Printf("Query: %s\n", query)
	output, err := s.sess.Query(query, format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", output)
}

func (s *chServer) handlePing(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Ok")
}

func (s *chServer) handlePlay(w http.ResponseWriter, r *http.Request) {
	data, err := content.ReadFile("play.html")
	if err != nil {
		http.Error(w, "Unable to open play.html", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, "play.html", time.Now(), bytes.NewReader(data))
}

func (s *chServer) handleNotFound(w http.ResponseWriter, r *http.Request) {
	data, err := content.ReadFile("play.html")
	if err != nil {
		http.Error(w, "Unable to open play.html", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, "play.html", time.Now(), bytes.NewReader(data))
}

func (s *chServer) tokenAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("X-API-Key")
		if s.auth.isTokenAuth {
			tokenHash := sha256.Sum256([]byte(authToken))
			expectedTokenHash := sha256.Sum256([]byte(s.auth.token))

			tokenMatch := (subtle.ConstantTimeCompare(tokenHash[:], expectedTokenHash[:]) == 1)

			if tokenMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (s *chServer) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(s.auth.username))
			expectedPasswordHash := sha256.Sum256([]byte(s.auth.password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

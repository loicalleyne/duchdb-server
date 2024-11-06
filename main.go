package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const (
	tokenAuth string = "token"
	userAuth  string = "user"
)

var port = flag.Int("port", 9999, "duckdb query API port")
var chPort = flag.Int("chport", 9998, "chdb query API port")
var authType = flag.String("authtype", tokenAuth, "API auth type - token||user")
var token = flag.String("token", "", "auth token")
var user = flag.String("user", "", "username")
var password = flag.String("password", "", "password")

func main() {
	var authString string
	flag.Parse()
	ctrlC()
	s := newServer(chDBpath)
	mux := http.NewServeMux()

	// Set authentication
	switch *authType {
	// Token to include in request header X-API-Key
	case tokenAuth:
		s.auth.token = *token
		s.auth.isTokenAuth = true
		// Register specific handlers
		mux.HandleFunc("/play", s.tokenAuth(s.handlePlay))
		mux.HandleFunc("/ping", s.tokenAuth(s.handlePing))
		mux.HandleFunc("/schema", s.tokenAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				s.handleSchemaPost(w, r)
				return
			}
			s.handleSchemaGet(w, r)
		}))

		// Catch-all handler: place this last
		mux.HandleFunc("/", s.tokenAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				if r.Method == http.MethodPost {
					s.handleRootPost(w, r)
					return
				}
				http.Redirect(w, r, "/play", http.StatusFound)
			default:
				s.handleNotFound(w, r) // Use this for 404 responses
			}
		}))
	// HTTP Basic Auth credentials
	case userAuth:
		if *user == "" {
			log.Fatal("No username provided")
		}
		authString = *user + ":" + *password
		s.auth.username = *user
		s.auth.password = *password
		// Register specific handlers
		mux.HandleFunc("/play", s.basicAuth(s.handlePlay))
		mux.HandleFunc("/ping", s.basicAuth(s.handlePing))
		mux.HandleFunc("/schema", s.basicAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				s.handleSchemaPost(w, r)
				return
			}
			s.handleSchemaGet(w, r)
		}))

		// Catch-all handler: place this last
		mux.HandleFunc("/", s.basicAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				if r.Method == http.MethodPost {
					s.handleRootPost(w, r)
					return
				}
				http.Redirect(w, r, "/play", http.StatusFound)
			default:
				s.handleNotFound(w, r) // Use this for 404 responses
			}
		}))
	default:
		log.Fatalf("Invalid auth type\nValid auth types:\n\t- token\n\t - user")
	}
	runHTTP(authString, mux)
}

// CtrlC intercepts any Ctrl+C keyboard input and exits to the shell.
func ctrlC() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Printf("Closing")
		fmt.Fprintf(os.Stdout, "🦆🏠 👋\n")
		os.Exit(2)
	}()
}

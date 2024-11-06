package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	duckdb "github.com/marcboeker/go-duckdb"
)

type DBPoolT struct {
	sync.Mutex
	db *sql.DB
}

// DB is the global database connection pool.
var DBPool DBPoolT

type rspBody struct {
	Count   *int                     `json:"count,omitempty"`
	Results []map[string]interface{} `json:"results,omitempty"`
	Error   string                   `json:"error,omitempty"`
}

func initDuckDB(dbPath string) error {
	var err error
	var db string
	if dbPath != "" {
		db = dbPath + "?access_mode=read_only&threads=4"
	}
	connector, err := duckdb.NewConnector(db, func(execer driver.ExecerContext) error {
		bootQueries := []string{
			"INSTALL 'json'",
			"LOAD 'json'",
			"INSTALL 'parquet'",
			"LOAD 'parquet'",
			"INSTALL 'httpfs'",
			"LOAD 'httpfs'",
			"INSTALL httpserver FROM community",
			"LOAD httpserver",
			"INSTALL shellfs from community",
			"LOAD shellfs",
			"INSTALL http_client FROM community",
			"LOAD http_client",
			// "INSTALL bigquery FROM community",
			// "LOAD bigquery",
		}

		for _, qry := range bootQueries {
			_, err = execer.ExecContext(context.Background(), qry, nil)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("connector error : %v\n", err)
		os.Exit(1)
	}

	// Create a new database pool with a maximum of 10 connections.
	DBPool.db = sql.OpenDB(connector)
	if err != nil {
		log.Fatal(err)
	}
	DBPool.db.SetMaxOpenConns(10)
	DBPool.db.SetMaxIdleConns(10)
	return nil
}

func runHTTP(authString string, mux *http.ServeMux) {
	db, err := DBPool.db.Conn(context.Background())
	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()
	query := fmt.Sprintf("SELECT httpserve_start('localhost', %v,'%v');", *port, authString)
	_, err = db.ExecContext(context.Background(), query)
	if err != nil {
		log.Fatal(err)
		return
	}
	go schemasTTL()
	log.Printf("🦆🏠 DuchDB HTTP Server started - duckdb file: %v - auth type: %v ", duckPath, *authType)
	log.Printf("Listening on :\n\tchdb - http://localhost:%d\n\tduckdb - http://localhost:%d\n", *chPort, *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *chPort), mux))
}

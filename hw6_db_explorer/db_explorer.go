package main

import (
	"database/sql"
	"net/http"
	"regexp"
)

var (
	tableRegexp       = regexp.MustCompile(`^\/[^\/]+\/?$`)
	tableWithIdRegexp = regexp.MustCompile(`^\/[^\/]+\/.+`)
)

type DbExplorer struct {
	db *sql.DB
}

func NewDbExplorer(db *sql.DB) (DbExplorer, error) {
	return DbExplorer{db: db}, nil
}

func (explorer DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET" && r.URL.Path == "/":
		explorer.getTables(w, r)
	case r.Method == "GET" && tableRegexp.MatchString(r.URL.Path):
		explorer.getRowsFromTable(w, r)
	case r.Method == "GET" && tableWithIdRegexp.MatchString(r.URL.Path):
		explorer.getRowFromTable(w, r)
	case r.Method == "PUT" && tableRegexp.MatchString(r.URL.Path):
		explorer.putRowToTable(w, r)
	case r.Method == "POST" && tableWithIdRegexp.MatchString(r.URL.Path):
		explorer.postRowInTable(w, r)
	case r.Method == "DELETE" && tableWithIdRegexp.MatchString(r.URL.Path):
		explorer.deleteRowFromTable(w, r)
	default:
		http.Error(w, "unknown method", http.StatusNotFound)
	}
}

// GET /
func (explorer DbExplorer) getTables(w http.ResponseWriter, r *http.Request) {
}

// GET /$table?limit=5&offset=7
func (explorer DbExplorer) getRowsFromTable(w http.ResponseWriter, r *http.Request) {
}

// GET /$table/$id
func (explorer DbExplorer) getRowFromTable(w http.ResponseWriter, r *http.Request) {
}

// PUT /$table
func (explorer DbExplorer) putRowToTable(w http.ResponseWriter, r *http.Request) {
}

// POST /$table/$id
func (explorer DbExplorer) postRowInTable(w http.ResponseWriter, r *http.Request) {
}

// DELETE /$table/$id
func (explorer DbExplorer) deleteRowFromTable(w http.ResponseWriter, r *http.Request) {
}

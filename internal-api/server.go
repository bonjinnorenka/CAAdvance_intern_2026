package main

import (
	"net/http"
	"time"
)

type apiServer struct {
	store     Store
	queue     reportQueue
	exportDir string
	now       func() time.Time
}

func (s *apiServer) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /me/ad_accounts", s.withUser(s.handleMeAdAccounts))
	mux.HandleFunc("GET /ad_accounts", s.withUser(s.handleAllAdAccounts))
	mux.HandleFunc("POST /report", s.withUser(s.handleCreateReport))
	mux.HandleFunc("GET /report", s.withUser(s.handleDownloadReport))
	mux.HandleFunc("GET /me/reports", s.withUser(s.handleMyReports))
	mux.HandleFunc("GET /users", s.withUser(s.handleListUsers))
	mux.HandleFunc("GET /user", s.withUser(s.handleGetUser))
	mux.HandleFunc("POST /user", s.withUser(s.handleCreateUser))
	mux.HandleFunc("PATCH /user", s.withUser(s.handleUpdateUser))
	mux.HandleFunc("DELETE /user", s.withUser(s.handleDeleteUser))
}

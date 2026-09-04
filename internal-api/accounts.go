package main

import (
	"log"
	"net/http"
)

func (s *apiServer) handleMeAdAccounts(w http.ResponseWriter, r *http.Request, user userRecord) {
	accounts, err := s.store.ListUserAdAccounts(r.Context(), user.ID)
	if err != nil {
		log.Printf("list user ad accounts: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}

func (s *apiServer) handleAllAdAccounts(w http.ResponseWriter, r *http.Request, user userRecord) {
	if !requireAdmin(w, user) {
		return
	}
	accounts, err := s.store.ListAllAdAccounts(r.Context())
	if err != nil {
		log.Printf("list all ad accounts: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}

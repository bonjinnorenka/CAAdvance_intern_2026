package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func (s *apiServer) handleListUsers(w http.ResponseWriter, r *http.Request, user userRecord) {
	if !requireAdmin(w, user) {
		return
	}
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		log.Printf("list users: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	out := make([]userSummaryResponse, 0, len(users))
	for _, u := range users {
		out = append(out, userSummaryResponse{
			ID:        u.ID,
			Name:      u.Name,
			Role:      u.Role,
			CreatedAt: formatDateTime(u.CreatedAt),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *apiServer) handleGetUser(w http.ResponseWriter, r *http.Request, user userRecord) {
	if !requireAdmin(w, user) {
		return
	}
	id, err := parseIDQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "id クエリが不正です")
		return
	}
	detail, accountIDs, err := s.store.GetUserDetail(r.Context(), id)
	if err == errNotFound {
		writeAPIError(w, http.StatusNotFound, "not_found", "ユーザーが見つかりません")
		return
	}
	if err != nil {
		log.Printf("get user detail %d: %v", id, err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	if accountIDs == nil {
		accountIDs = []string{}
	}
	writeJSON(w, http.StatusOK, userDetailResponse{
		ID:           detail.ID,
		Name:         detail.Name,
		Role:         detail.Role,
		CreatedAt:    formatDateTime(detail.CreatedAt),
		UpdatedAt:    formatDateTime(detail.UpdatedAt),
		AdAccountIDs: accountIDs,
	})
}

func (s *apiServer) handleCreateUser(w http.ResponseWriter, r *http.Request, user userRecord) {
	if !requireAdmin(w, user) {
		return
	}
	var body userCreateRequest
	if err := decodeJSON(r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "リクエストボディが不正です")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "name は必須です")
		return
	}
	if tooLong(body.Name, maxNameLen) {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "name は50文字以内です")
		return
	}
	if !validRole(body.Role) {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "role が不正です")
		return
	}
	if body.AdAccountIDs == nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "ad_account_ids は必須です")
		return
	}
	if !s.ensureAdAccountsExist(w, r, body.AdAccountIDs) {
		return
	}

	if _, err := s.store.CreateUser(r.Context(), userCreateInput{
		Name:         body.Name,
		Role:         body.Role,
		AdAccountIDs: body.AdAccountIDs,
	}); err != nil {
		log.Printf("create user: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *apiServer) handleUpdateUser(w http.ResponseWriter, r *http.Request, user userRecord) {
	if !requireAdmin(w, user) {
		return
	}
	id, err := parseIDQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "id クエリが不正です")
		return
	}
	var body userUpdateRequest
	if err := decodeJSON(r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "リクエストボディが不正です")
		return
	}
	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if name == "" {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "name は空にできません")
			return
		}
		if tooLong(name, maxNameLen) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "name は50文字以内です")
			return
		}
		body.Name = &name
	}
	if body.Role != nil && !validRole(*body.Role) {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "role が不正です")
		return
	}
	if body.AdAccountIDs != nil && !s.ensureAdAccountsExist(w, r, *body.AdAccountIDs) {
		return
	}

	updated, accountIDs, err := s.store.UpdateUser(r.Context(), id, userUpdateInput{
		Name:         body.Name,
		Role:         body.Role,
		AdAccountIDs: body.AdAccountIDs,
	})
	if err == errNotFound {
		writeAPIError(w, http.StatusNotFound, "not_found", "ユーザーが見つかりません")
		return
	}
	if err != nil {
		log.Printf("update user %d: %v", id, err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	if accountIDs == nil {
		accountIDs = []string{}
	}
	writeJSON(w, http.StatusOK, userUpdateResponse{
		ID:           updated.ID,
		Name:         updated.Name,
		Role:         updated.Role,
		AdAccountIDs: accountIDs,
	})
}

func (s *apiServer) handleDeleteUser(w http.ResponseWriter, r *http.Request, user userRecord) {
	if !requireAdmin(w, user) {
		return
	}
	id, err := parseIDQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "id クエリが不正です")
		return
	}
	if err := s.store.SoftDeleteUser(r.Context(), id); err == errNotFound {
		writeAPIError(w, http.StatusNotFound, "not_found", "ユーザーが見つかりません")
		return
	} else if err != nil {
		log.Printf("delete user %d: %v", id, err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *apiServer) ensureAdAccountsExist(w http.ResponseWriter, r *http.Request, accountIDs []string) bool {
	missing, err := s.store.MissingAdAccountIDs(r.Context(), accountIDs)
	if err != nil {
		log.Printf("validate ad accounts: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return false
	}
	if len(missing) > 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_request",
			fmt.Sprintf("存在しない広告アカウントが含まれています: %s", strings.Join(missing, ", ")))
		return false
	}
	return true
}

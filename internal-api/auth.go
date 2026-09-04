package main

import (
	"log"
	"net/http"
	"strconv"
)

type userHandler func(http.ResponseWriter, *http.Request, userRecord)

func (s *apiServer) withUser(next userHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-User-Id")
		if raw == "" {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "X-User-Id ヘッダーが必要です")
			return
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "X-User-Id が不正です")
			return
		}
		user, err := s.store.GetUser(r.Context(), id)
		if err == errNotFound {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "ユーザーが見つかりません")
			return
		}
		if err != nil {
			log.Printf("get user %d: %v", id, err)
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
			return
		}
		next(w, r, user)
	}
}

func requireAdmin(w http.ResponseWriter, user userRecord) bool {
	if user.Role != roleAdmin {
		writeAPIError(w, http.StatusForbidden, "forbidden", "ユーザー管理権限がありません")
		return false
	}
	return true
}

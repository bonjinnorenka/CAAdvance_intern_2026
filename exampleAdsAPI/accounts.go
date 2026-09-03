package main

import (
	"net/http"
	"strconv"
)

type account struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}

type accountsResponse struct {
	Data  []account `json:"data"`
	Page  int       `json:"page"`
	Total int       `json:"total"`
}

var accounts = []account{
	{AccountID: "acc_00101", AccountName: "サンプル商事", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00102", AccountName: "テスト物産", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00103", AccountName: "デモ広告社", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00104", AccountName: "架空フーズ", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00105", AccountName: "仮想トラベル", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00106", AccountName: "例示アパレル", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00107", AccountName: "モック電機", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00108", AccountName: "仮設ホーム", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00109", AccountName: "試作コスメ", Currency: "JPY", Status: "active"},
	{AccountID: "acc_00110", AccountName: "演習メディア", Currency: "JPY", Status: "active"},
}

func findAccount(id string) (account, bool) {
	for _, a := range accounts {
		if a.AccountID == id {
			return a, true
		}
	}
	return account{}, false
}

func parseBoundedInt(raw, name string, defaultVal, min, max int) (int, string) {
	if raw == "" {
		return defaultVal, ""
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, name + " は整数で指定してください"
	}
	if n < min || (max > 0 && n > max) {
		if max > 0 {
			return 0, name + " は " + strconv.Itoa(min) + " 以上 " + strconv.Itoa(max) + " 以下の整数で指定してください"
		}
		return 0, name + " は " + strconv.Itoa(min) + " 以上の整数で指定してください"
	}
	return n, ""
}

func (s *server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, msg := parseBoundedInt(q.Get("page"), "page", 1, 1, 0)
	if msg != "" {
		writeError(w, http.StatusBadRequest, "invalid_request", msg)
		return
	}
	limit, msg := parseBoundedInt(q.Get("limit"), "limit", 100, 1, 100)
	if msg != "" {
		writeError(w, http.StatusBadRequest, "invalid_request", msg)
		return
	}

	start := (page - 1) * limit
	data := []account{}
	if start < len(accounts) {
		end := start + limit
		if end > len(accounts) {
			end = len(accounts)
		}
		data = accounts[start:end]
	}

	writeJSON(w, http.StatusOK, accountsResponse{
		Data:  data,
		Page:  page,
		Total: len(accounts),
	})
}

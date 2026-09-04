package main

import "time"

type userRecord struct {
	ID        int64
	Name      string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type adAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type reportRecord struct {
	ID        int64
	Name      string
	Status    string
	Reason    *string
	CreatedAt time.Time
	FilePath  string
	CreatedBy int64
	DateFrom  time.Time
	DateTo    time.Time
}

type reportInsert struct {
	Name         string
	CreatedBy    int64
	CreatedAt    time.Time
	DateFrom     time.Time
	DateTo       time.Time
	MarginRate   int
	AdAccountIDs []string
}

type userCreateInput struct {
	Name         string
	Role         string
	AdAccountIDs []string
}

type userUpdateInput struct {
	Name         *string
	Role         *string
	AdAccountIDs *[]string
}

type userSummaryResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type userDetailResponse struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	AdAccountIDs []string `json:"ad_account_ids"`
}

type userUpdateResponse struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	AdAccountIDs []string `json:"ad_account_ids"`
}

type reportResponse struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Reason    *string `json:"reason"`
	CreatedAt string  `json:"created_at"`
}

type reportCreateRequest struct {
	AdAccountIDs []string `json:"ad_account_ids"`
	DateFrom     string   `json:"date_from"`
	DateTo       string   `json:"date_to"`
	MarginRate   int      `json:"margin_rate"`
}

type reportCreateAccepted struct {
	JobID  int64  `json:"job_id"`
	Status string `json:"status"`
}

type userCreateRequest struct {
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	AdAccountIDs []string `json:"ad_account_ids"`
}

type userUpdateRequest struct {
	Name         *string   `json:"name"`
	Role         *string   `json:"role"`
	AdAccountIDs *[]string `json:"ad_account_ids"`
}

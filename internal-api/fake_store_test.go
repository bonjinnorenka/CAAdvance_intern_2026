package main

import (
	"context"
	"sync"
	"time"
)

type fakeStore struct {
	mu         sync.Mutex
	users      map[int64]userRecord
	deleted    map[int64]bool
	accounts   map[string]adAccount
	perms      map[int64][]string
	reports    map[int64]reportRecord
	nextUser   int64
	nextReport int64
}

func newFakeStore() *fakeStore {
	now := time.Date(2026, 8, 10, 13, 30, 0, 0, jst)
	return &fakeStore{
		users: map[int64]userRecord{
			1: {ID: 1, Name: "管理者", Role: roleAdmin, CreatedAt: now, UpdatedAt: now},
			2: {ID: 2, Name: "一般ユーザー", Role: roleMember, CreatedAt: now, UpdatedAt: now},
		},
		deleted: map[int64]bool{},
		accounts: map[string]adAccount{
			"acc_00101": {ID: "acc_00101", Name: "A社"},
			"acc_00102": {ID: "acc_00102", Name: "B社"},
			"acc_00106": {ID: "acc_00106", Name: "F社"},
			"acc_00109": {ID: "acc_00109", Name: "I社"},
		},
		perms: map[int64][]string{
			1: {"acc_00101", "acc_00102"},
			2: {"acc_00106"},
		},
		reports:    map[int64]reportRecord{},
		nextUser:   3,
		nextReport: 1,
	}
}

func (s *fakeStore) GetUser(_ context.Context, id int64) (userRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleted[id] {
		return userRecord{}, errNotFound
	}
	user, ok := s.users[id]
	if !ok {
		return userRecord{}, errNotFound
	}
	return user, nil
}

func (s *fakeStore) ListUsers(_ context.Context) ([]userRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]userRecord, 0)
	for id := int64(1); id < s.nextUser; id++ {
		if s.deleted[id] {
			continue
		}
		if user, ok := s.users[id]; ok {
			out = append(out, user)
		}
	}
	return out, nil
}

func (s *fakeStore) GetUserDetail(_ context.Context, id int64) (userRecord, []string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleted[id] {
		return userRecord{}, nil, errNotFound
	}
	user, ok := s.users[id]
	if !ok {
		return userRecord{}, nil, errNotFound
	}
	ids := append([]string{}, s.perms[id]...)
	return user, ids, nil
}

func (s *fakeStore) CreateUser(_ context.Context, in userCreateInput) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextUser
	s.nextUser++
	now := time.Now().In(jst)
	s.users[id] = userRecord{ID: id, Name: in.Name, Role: in.Role, CreatedAt: now, UpdatedAt: now}
	s.perms[id] = uniqueStrings(in.AdAccountIDs)
	return id, nil
}

func (s *fakeStore) UpdateUser(_ context.Context, id int64, in userUpdateInput) (userRecord, []string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleted[id] {
		return userRecord{}, nil, errNotFound
	}
	user, ok := s.users[id]
	if !ok {
		return userRecord{}, nil, errNotFound
	}
	if in.Name != nil {
		user.Name = *in.Name
	}
	if in.Role != nil {
		user.Role = *in.Role
	}
	user.UpdatedAt = time.Now().In(jst)
	s.users[id] = user
	if in.AdAccountIDs != nil {
		s.perms[id] = uniqueStrings(*in.AdAccountIDs)
	}
	ids := append([]string{}, s.perms[id]...)
	return user, ids, nil
}

func (s *fakeStore) SoftDeleteUser(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleted[id] {
		return errNotFound
	}
	if _, ok := s.users[id]; !ok {
		return errNotFound
	}
	s.deleted[id] = true
	return nil
}

func (s *fakeStore) ListUserAdAccounts(_ context.Context, userID int64) ([]adAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]adAccount, 0)
	for _, id := range s.perms[userID] {
		if acc, ok := s.accounts[id]; ok {
			out = append(out, acc)
		}
	}
	return out, nil
}

func (s *fakeStore) ListAllAdAccounts(_ context.Context) ([]adAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]adAccount, 0, len(s.accounts))
	for _, id := range []string{"acc_00101", "acc_00102", "acc_00106", "acc_00109"} {
		if acc, ok := s.accounts[id]; ok {
			out = append(out, acc)
		}
	}
	return out, nil
}

func (s *fakeStore) UnauthorizedAccountIDs(_ context.Context, userID int64, accountIDs []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	allowed := make(map[string]struct{})
	for _, id := range s.perms[userID] {
		if _, ok := s.accounts[id]; ok {
			allowed[id] = struct{}{}
		}
	}
	return missingFrom(accountIDs, allowed), nil
}

func (s *fakeStore) MissingAdAccountIDs(_ context.Context, accountIDs []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := make(map[string]struct{}, len(s.accounts))
	for id := range s.accounts {
		found[id] = struct{}{}
	}
	return missingFrom(accountIDs, found), nil
}

func (s *fakeStore) CreateQueuedReport(_ context.Context, in reportInsert) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextReport
	s.nextReport++
	s.reports[id] = reportRecord{
		ID:        id,
		Name:      in.Name,
		Status:    "queued",
		CreatedAt: in.CreatedAt,
		CreatedBy: in.CreatedBy,
	}
	return id, nil
}

func (s *fakeStore) MarkReportFailed(_ context.Context, id int64, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.reports[id]
	if !ok {
		return errNotFound
	}
	rec.Status = "failed"
	rec.Reason = &reason
	s.reports[id] = rec
	return nil
}

func (s *fakeStore) GetReport(_ context.Context, id int64) (reportRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.reports[id]
	if !ok {
		return reportRecord{}, errNotFound
	}
	return rec, nil
}

func (s *fakeStore) ListReportsByUser(_ context.Context, userID int64) ([]reportRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]reportRecord, 0)
	for id := s.nextReport - 1; id >= 1; id-- {
		rec, ok := s.reports[id]
		if !ok || rec.CreatedBy != userID {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

type fakeQueue struct {
	mu   sync.Mutex
	jobs []int64
	err  error
}

func (q *fakeQueue) EnqueueGenerate(_ context.Context, reportID int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.err != nil {
		return q.err
	}
	q.jobs = append(q.jobs, reportID)
	return nil
}

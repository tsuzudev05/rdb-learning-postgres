// Package mock provides in-memory mock implementations of domain repositories
// for use in unit tests. No external dependencies required.
package mock

import (
	"context"
	"fmt"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/keyresult"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// ─── UserRepository ──────────────────────────────────────────────────────────

// UserRepository is an in-memory mock for repository.UserRepository.
type UserRepository struct {
	Users map[string]user.User // key: UserId.Value()
	// ErrFindByEmail is returned by FindByEmail when set (simulates DB error).
	ErrFindByEmail error
	ErrSave        error
}

// NewUserRepository initialises an empty mock.
func NewUserRepository() *UserRepository {
	return &UserRepository{Users: make(map[string]user.User)}
}

func (r *UserRepository) FindByID(_ context.Context, id user.UserId) (*user.User, error) {
	u, ok := r.Users[id.Value()]
	if !ok {
		return nil, nil
	}
	return &u, nil
}

func (r *UserRepository) FindByEmail(_ context.Context, email user.Email) (*user.User, error) {
	if r.ErrFindByEmail != nil {
		return nil, r.ErrFindByEmail
	}
	for _, u := range r.Users {
		if u.Email().Equal(email) {
			return &u, nil
		}
	}
	return nil, nil
}

func (r *UserRepository) FindAll(_ context.Context) ([]user.User, error) {
	users := make([]user.User, 0, len(r.Users))
	for _, u := range r.Users {
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) Save(_ context.Context, u user.User) error {
	if r.ErrSave != nil {
		return r.ErrSave
	}
	r.Users[u.ID().Value()] = u
	return nil
}

func (r *UserRepository) Remove(_ context.Context, id user.UserId) error {
	delete(r.Users, id.Value())
	return nil
}

// ─── TeamRepository ──────────────────────────────────────────────────────────

// TeamRepository is an in-memory mock for repository.TeamRepository.
type TeamRepository struct {
	Teams   map[string]team.Team // key: TeamId.Value()
	ErrSave error
}

// NewTeamRepository initialises an empty mock.
func NewTeamRepository() *TeamRepository {
	return &TeamRepository{Teams: make(map[string]team.Team)}
}

func (r *TeamRepository) FindByID(_ context.Context, id team.TeamId) (*team.Team, error) {
	t, ok := r.Teams[id.Value()]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

func (r *TeamRepository) FindByUserID(_ context.Context, uid user.UserId) ([]team.Team, error) {
	var result []team.Team
	for _, t := range r.Teams {
		for _, m := range t.Members() {
			if m.UserID().Equal(uid) {
				result = append(result, t)
				break
			}
		}
	}
	return result, nil
}

func (r *TeamRepository) FindAll(_ context.Context) ([]team.Team, error) {
	teams := make([]team.Team, 0, len(r.Teams))
	for _, t := range r.Teams {
		teams = append(teams, t)
	}
	return teams, nil
}

func (r *TeamRepository) Save(_ context.Context, t team.Team) error {
	if r.ErrSave != nil {
		return r.ErrSave
	}
	r.Teams[t.ID().Value()] = t
	return nil
}

func (r *TeamRepository) Remove(_ context.Context, id team.TeamId) error {
	delete(r.Teams, id.Value())
	return nil
}

// ─── KeyResultRepository ─────────────────────────────────────────────────────

// KeyResultRepository is an in-memory mock for repository.KeyResultRepository.
type KeyResultRepository struct {
	KeyResults map[string]keyresult.KeyResult // key: KeyResultId.Value()
	ErrSave    error
	ErrFindByID error
}

// NewKeyResultRepository initialises an empty mock.
func NewKeyResultRepository() *KeyResultRepository {
	return &KeyResultRepository{KeyResults: make(map[string]keyresult.KeyResult)}
}

func (r *KeyResultRepository) FindByID(_ context.Context, id keyresult.KeyResultId) (*keyresult.KeyResult, error) {
	if r.ErrFindByID != nil {
		return nil, r.ErrFindByID
	}
	kr, ok := r.KeyResults[id.Value()]
	if !ok {
		return nil, nil
	}
	return &kr, nil
}

func (r *KeyResultRepository) FindByObjectiveID(_ context.Context, objectiveId objective.ObjectiveId) ([]keyresult.KeyResult, error) {
	var result []keyresult.KeyResult
	for _, kr := range r.KeyResults {
		if kr.ObjectiveID().Equal(objectiveId) {
			result = append(result, kr)
		}
	}
	return result, nil
}

func (r *KeyResultRepository) FindByOwnerID(_ context.Context, ownerId user.UserId) ([]keyresult.KeyResult, error) {
	var result []keyresult.KeyResult
	for _, kr := range r.KeyResults {
		if kr.OwnerID().Equal(ownerId) {
			result = append(result, kr)
		}
	}
	return result, nil
}

func (r *KeyResultRepository) Save(_ context.Context, kr keyresult.KeyResult) error {
	if r.ErrSave != nil {
		return r.ErrSave
	}
	r.KeyResults[kr.ID().Value()] = kr
	return nil
}

func (r *KeyResultRepository) Remove(_ context.Context, id keyresult.KeyResultId) error {
	delete(r.KeyResults, id.Value())
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// ErrDB is a generic sentinel error for simulating DB failures in tests.
var ErrDB = fmt.Errorf("mock: DB接続エラー")

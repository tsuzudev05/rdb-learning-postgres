package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

// TeamUseCase handles application logic for the Team aggregate.
type TeamUseCase struct {
	repo repository.TeamRepository
}

// NewTeamUseCase creates a TeamUseCase with the given repository.
func NewTeamUseCase(repo repository.TeamRepository) *TeamUseCase {
	return &TeamUseCase{repo: repo}
}

// CreateTeamInput holds the input data for creating a team.
type CreateTeamInput struct {
	Name        string
	Description *string
}

// CreateTeam creates and persists a new team.
func (uc *TeamUseCase) CreateTeam(ctx context.Context, in CreateTeamInput) (*team.Team, error) {
	tid, err := team.NewTeamId(uuid.New().String())
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase.CreateTeam: UUID生成エラー: %w", err)
	}

	now := time.Now().UTC()
	t, err := team.NewTeam(tid, in.Name, in.Description, nil, now, now)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("TeamUseCase.CreateTeam: 保存エラー: %w", err)
	}
	return &t, nil
}

// GetTeam returns the team with the given ID, or (nil, nil) if not found.
func (uc *TeamUseCase) GetTeam(ctx context.Context, id team.TeamId) (*team.Team, error) {
	t, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase.GetTeam: %w", err)
	}
	return t, nil
}

// AddMemberInput holds the input data for adding a team member.
type AddMemberInput struct {
	TeamID string
	UserID string
	Role   string // "admin" | "member"
}

// AddMember performs Read-Modify-Write to add a member to the team.
// Returns error if the team is not found or the user is already a member.
func (uc *TeamUseCase) AddMember(ctx context.Context, in AddMemberInput) (*team.Team, error) {
	tid, err := team.NewTeamId(in.TeamID)
	if err != nil {
		return nil, err
	}
	uid, err := user.NewUserId(in.UserID)
	if err != nil {
		return nil, err
	}
	roleStr := in.Role
	if roleStr == "" {
		roleStr = "member"
	}
	role, err := team.NewRole(roleStr)
	if err != nil {
		return nil, err
	}

	t, err := uc.repo.FindByID(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase.AddMember: チーム取得エラー: %w", err)
	}
	if t == nil {
		return nil, fmt.Errorf("TeamUseCase.AddMember: チームが見つかりません: %s", in.TeamID)
	}

	mid, err := team.NewTeamMemberId(uuid.New().String())
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase.AddMember: UUID生成エラー: %w", err)
	}
	member, err := team.NewTeamMember(mid, tid, uid, role, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := t.AddMember(member); err != nil {
		return nil, fmt.Errorf("TeamUseCase.AddMember: %w", err)
	}

	if err := uc.repo.Save(ctx, *t); err != nil {
		return nil, fmt.Errorf("TeamUseCase.AddMember: 保存エラー: %w", err)
	}
	return t, nil
}

// RemoveMemberInput holds the input data for removing a team member.
type RemoveMemberInput struct {
	TeamID string
	UserID string
}

// RemoveMember performs Read-Modify-Write to remove a member from the team.
// No-op if the user is not a member.
func (uc *TeamUseCase) RemoveMember(ctx context.Context, in RemoveMemberInput) (*team.Team, error) {
	tid, err := team.NewTeamId(in.TeamID)
	if err != nil {
		return nil, err
	}
	uid, err := user.NewUserId(in.UserID)
	if err != nil {
		return nil, err
	}

	t, err := uc.repo.FindByID(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase.RemoveMember: チーム取得エラー: %w", err)
	}
	if t == nil {
		return nil, fmt.Errorf("TeamUseCase.RemoveMember: チームが見つかりません: %s", in.TeamID)
	}

	t.RemoveMember(uid)

	if err := uc.repo.Save(ctx, *t); err != nil {
		return nil, fmt.Errorf("TeamUseCase.RemoveMember: 保存エラー: %w", err)
	}
	return t, nil
}

// ChangeMemberRoleInput holds the input data for changing a member's role.
type ChangeMemberRoleInput struct {
	TeamID string
	UserID string
	Role   string // "admin" | "member"
}

// ChangeMemberRole performs Read-Modify-Write to change a member's role.
// Returns error if the user is not a member of the team.
func (uc *TeamUseCase) ChangeMemberRole(ctx context.Context, in ChangeMemberRoleInput) (*team.Team, error) {
	tid, err := team.NewTeamId(in.TeamID)
	if err != nil {
		return nil, err
	}
	uid, err := user.NewUserId(in.UserID)
	if err != nil {
		return nil, err
	}
	role, err := team.NewRole(in.Role)
	if err != nil {
		return nil, err
	}

	t, err := uc.repo.FindByID(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase.ChangeMemberRole: チーム取得エラー: %w", err)
	}
	if t == nil {
		return nil, fmt.Errorf("TeamUseCase.ChangeMemberRole: チームが見つかりません: %s", in.TeamID)
	}

	if err := t.ChangeMemberRole(uid, role); err != nil {
		return nil, fmt.Errorf("TeamUseCase.ChangeMemberRole: %w", err)
	}

	if err := uc.repo.Save(ctx, *t); err != nil {
		return nil, fmt.Errorf("TeamUseCase.ChangeMemberRole: 保存エラー: %w", err)
	}
	return t, nil
}

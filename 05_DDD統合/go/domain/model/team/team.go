package team

import (
	"fmt"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// Team is the aggregate root for the Team aggregate.
// Members are managed inside the aggregate (Read-Modify-Write via Repository).
type Team struct {
	id          TeamId
	name        string
	description *string // optional
	members     []TeamMember
	createdAt   time.Time
	updatedAt   time.Time
}

func NewTeam(
	id TeamId,
	name string,
	description *string,
	members []TeamMember,
	createdAt time.Time,
	updatedAt time.Time,
) (Team, error) {
	if name == "" {
		return Team{}, fmt.Errorf("Team: チーム名は空にできません")
	}
	m := members
	if m == nil {
		m = []TeamMember{}
	}
	return Team{
		id:          id,
		name:        name,
		description: description,
		members:     m,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

func (t Team) ID() TeamId            { return t.id }
func (t Team) Name() string          { return t.name }
func (t Team) Description() *string  { return t.description }
func (t Team) Members() []TeamMember { return t.members }
func (t Team) CreatedAt() time.Time  { return t.createdAt }
func (t Team) UpdatedAt() time.Time  { return t.updatedAt }

// AddMember appends a member. Returns error if the user is already a member.
func (t *Team) AddMember(m TeamMember) error {
	for _, existing := range t.members {
		if existing.UserID().Equal(m.UserID()) {
			return fmt.Errorf("Team.AddMember: ユーザー %s は既にメンバーです", m.UserID())
		}
	}
	t.members = append(t.members, m)
	return nil
}

// RemoveMember removes the member with the given userId. No-op if not found.
func (t *Team) RemoveMember(uid user.UserId) {
	filtered := t.members[:0]
	for _, m := range t.members {
		if !m.UserID().Equal(uid) {
			filtered = append(filtered, m)
		}
	}
	t.members = filtered
}

// ChangeMemberRole updates the role of the member with the given userId.
func (t *Team) ChangeMemberRole(uid user.UserId, r Role) error {
	for i, m := range t.members {
		if m.UserID().Equal(uid) {
			t.members[i] = m.WithRole(r)
			return nil
		}
	}
	return fmt.Errorf("Team.ChangeMemberRole: ユーザー %s はメンバーではありません", uid)
}

package team

import (
	"fmt"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// TeamMember is an entity within the Team aggregate.
type TeamMember struct {
	id       TeamMemberId
	teamId   TeamId
	userId   user.UserId
	role     Role
	joinedAt time.Time
}

func NewTeamMember(
	id TeamMemberId,
	teamId TeamId,
	userId user.UserId,
	role Role,
	joinedAt time.Time,
) (TeamMember, error) {
	if (TeamMemberId{}) == id {
		return TeamMember{}, fmt.Errorf("TeamMember: id は必須です")
	}
	return TeamMember{
		id:       id,
		teamId:   teamId,
		userId:   userId,
		role:     role,
		joinedAt: joinedAt,
	}, nil
}

func (m TeamMember) ID() TeamMemberId { return m.id }
func (m TeamMember) TeamID() TeamId   { return m.teamId }
func (m TeamMember) UserID() user.UserId { return m.userId }
func (m TeamMember) Role() Role       { return m.role }
func (m TeamMember) JoinedAt() time.Time { return m.joinedAt }

// WithRole returns a new TeamMember with the role changed.
func (m TeamMember) WithRole(r Role) TeamMember {
	m.role = r
	return m
}

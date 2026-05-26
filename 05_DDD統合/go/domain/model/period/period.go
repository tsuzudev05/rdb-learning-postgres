package period

import (
	"fmt"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
)

// Period is the aggregate root for a half-year OKR cycle.
type Period struct {
	id        PeriodId
	teamId    team.TeamId
	name      string
	half      Half
	dateRange DateRange
	createdAt time.Time
	updatedAt time.Time
}

func NewPeriod(
	id PeriodId,
	teamId team.TeamId,
	name string,
	half Half,
	dateRange DateRange,
	createdAt time.Time,
	updatedAt time.Time,
) (Period, error) {
	if name == "" {
		return Period{}, fmt.Errorf("Period: 期間名は空にできません")
	}
	return Period{
		id:        id,
		teamId:    teamId,
		name:      name,
		half:      half,
		dateRange: dateRange,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

func (p Period) ID() PeriodId         { return p.id }
func (p Period) TeamID() team.TeamId  { return p.teamId }
func (p Period) Name() string         { return p.name }
func (p Period) Half() Half           { return p.half }
func (p Period) DateRange() DateRange { return p.dateRange }
func (p Period) CreatedAt() time.Time { return p.createdAt }
func (p Period) UpdatedAt() time.Time { return p.updatedAt }

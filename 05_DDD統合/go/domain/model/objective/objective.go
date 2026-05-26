package objective

import (
	"fmt"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// Objective is the aggregate root for the Objective (O in OKR).
type Objective struct {
	id           ObjectiveId
	periodId     period.PeriodId
	ownerId      user.UserId
	title        string
	description  *string // optional
	displayOrder int
	createdAt    time.Time
	updatedAt    time.Time
}

func NewObjective(
	id ObjectiveId,
	periodId period.PeriodId,
	ownerId user.UserId,
	title string,
	description *string,
	displayOrder int,
	createdAt time.Time,
	updatedAt time.Time,
) (Objective, error) {
	if title == "" {
		return Objective{}, fmt.Errorf("Objective: タイトルは空にできません")
	}
	if displayOrder < 0 {
		return Objective{}, fmt.Errorf("Objective: displayOrder は 0 以上でなければなりません")
	}
	return Objective{
		id:           id,
		periodId:     periodId,
		ownerId:      ownerId,
		title:        title,
		description:  description,
		displayOrder: displayOrder,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}, nil
}

func (o Objective) ID() ObjectiveId        { return o.id }
func (o Objective) PeriodID() period.PeriodId { return o.periodId }
func (o Objective) OwnerID() user.UserId   { return o.ownerId }
func (o Objective) Title() string          { return o.title }
func (o Objective) Description() *string   { return o.description }
func (o Objective) DisplayOrder() int      { return o.displayOrder }
func (o Objective) CreatedAt() time.Time   { return o.createdAt }
func (o Objective) UpdatedAt() time.Time   { return o.updatedAt }

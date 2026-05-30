package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

// ObjectiveHandler handles /api/v1/objectives endpoints.
type ObjectiveHandler struct {
	repo repository.ObjectiveRepository
}

// NewObjectiveHandler creates an ObjectiveHandler with the given repository.
func NewObjectiveHandler(repo repository.ObjectiveRepository) *ObjectiveHandler {
	return &ObjectiveHandler{repo: repo}
}

// ── リクエスト / レスポンス DTO ────────────────────────────────────────────────

type createObjectiveRequest struct {
	PeriodID     string  `json:"period_id"`
	OwnerID      string  `json:"owner_id"`
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	DisplayOrder int     `json:"display_order"`
}

type objectiveResponse struct {
	ID           string  `json:"id"`
	PeriodID     string  `json:"period_id"`
	OwnerID      string  `json:"owner_id"`
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	DisplayOrder int     `json:"display_order"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func toObjectiveResponse(o objective.Objective) objectiveResponse {
	return objectiveResponse{
		ID:           o.ID().Value(),
		PeriodID:     o.PeriodID().Value(),
		OwnerID:      o.OwnerID().Value(),
		Title:        o.Title(),
		Description:  o.Description(),
		DisplayOrder: o.DisplayOrder(),
		CreatedAt:    o.CreatedAt().Format(time.RFC3339),
		UpdatedAt:    o.UpdatedAt().Format(time.RFC3339),
	}
}

// ── ハンドラー実装 ─────────────────────────────────────────────────────────────

// List handles GET /api/v1/objectives
// クエリパラメータ: period_id または owner_id（省略時は全件）
func (h *ObjectiveHandler) List(c echo.Context) error {
	ctx := c.Request().Context()

	if periodIDStr := c.QueryParam("period_id"); periodIDStr != "" {
		pid, err := period.NewPeriodId(periodIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		objectives, err := h.repo.FindByPeriodID(ctx, pid)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		res := make([]objectiveResponse, len(objectives))
		for i, o := range objectives {
			res[i] = toObjectiveResponse(o)
		}
		return c.JSON(http.StatusOK, res)
	}

	if ownerIDStr := c.QueryParam("owner_id"); ownerIDStr != "" {
		uid, err := user.NewUserId(ownerIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		objectives, err := h.repo.FindByOwnerID(ctx, uid)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		res := make([]objectiveResponse, len(objectives))
		for i, o := range objectives {
			res[i] = toObjectiveResponse(o)
		}
		return c.JSON(http.StatusOK, res)
	}

	// FindAll はインターフェースに存在しないため period_id または owner_id を必須とする
	return echo.NewHTTPError(http.StatusBadRequest, "period_id または owner_id クエリパラメータを指定してください")
}

// Create handles POST /api/v1/objectives
func (h *ObjectiveHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var req createObjectiveRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストのパースに失敗しました: "+err.Error())
	}
	if req.PeriodID == "" || req.OwnerID == "" || req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "period_id, owner_id, title は必須です")
	}

	pid, err := period.NewPeriodId(req.PeriodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	uid, err := user.NewUserId(req.OwnerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	oid, err := objective.NewObjectiveId(newUUID())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	now := time.Now().UTC()
	o, err := objective.NewObjective(oid, pid, uid, req.Title, req.Description, req.DisplayOrder, now, now)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.repo.Save(ctx, o); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, toObjectiveResponse(o))
}

// Get handles GET /api/v1/objectives/:id
func (h *ObjectiveHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()

	oid, err := objective.NewObjectiveId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	o, err := h.repo.FindByID(ctx, oid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if o == nil {
		return echo.NewHTTPError(http.StatusNotFound, "目標が見つかりません")
	}
	return c.JSON(http.StatusOK, toObjectiveResponse(*o))
}

// Delete handles DELETE /api/v1/objectives/:id
func (h *ObjectiveHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	oid, err := objective.NewObjectiveId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.repo.Remove(ctx, oid); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

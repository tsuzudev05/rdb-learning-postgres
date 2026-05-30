package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

// PeriodHandler handles /api/v1/periods endpoints.
type PeriodHandler struct {
	repo repository.PeriodRepository
}

// NewPeriodHandler creates a PeriodHandler with the given repository.
func NewPeriodHandler(repo repository.PeriodRepository) *PeriodHandler {
	return &PeriodHandler{repo: repo}
}

// ── リクエスト / レスポンス DTO ────────────────────────────────────────────────

type createPeriodRequest struct {
	TeamID    string `json:"team_id"`
	Name      string `json:"name"`
	Half      string `json:"half"`       // "H1" | "H2"
	StartDate string `json:"start_date"` // "YYYY-MM-DD"
	EndDate   string `json:"end_date"`   // "YYYY-MM-DD"
}

type periodResponse struct {
	ID        string `json:"id"`
	TeamID    string `json:"team_id"`
	Name      string `json:"name"`
	Half      string `json:"half"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toPeriodResponse(p period.Period) periodResponse {
	return periodResponse{
		ID:        p.ID().Value(),
		TeamID:    p.TeamID().Value(),
		Name:      p.Name(),
		Half:      p.Half().Value(),
		StartDate: p.DateRange().StartString(),
		EndDate:   p.DateRange().EndString(),
		CreatedAt: p.CreatedAt().Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt().Format(time.RFC3339),
	}
}

// ── ハンドラー実装 ─────────────────────────────────────────────────────────────

// List handles GET /api/v1/periods
// クエリパラメータ: team_id（省略時は全件）
func (h *PeriodHandler) List(c echo.Context) error {
	ctx := c.Request().Context()

	if teamIDStr := c.QueryParam("team_id"); teamIDStr != "" {
		tid, err := team.NewTeamId(teamIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		periods, err := h.repo.FindByTeamID(ctx, tid)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		res := make([]periodResponse, len(periods))
		for i, p := range periods {
			res[i] = toPeriodResponse(p)
		}
		return c.JSON(http.StatusOK, res)
	}

	periods, err := h.repo.FindAll(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	res := make([]periodResponse, len(periods))
	for i, p := range periods {
		res[i] = toPeriodResponse(p)
	}
	return c.JSON(http.StatusOK, res)
}

// Create handles POST /api/v1/periods
func (h *PeriodHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var req createPeriodRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストのパースに失敗しました: "+err.Error())
	}
	if req.TeamID == "" || req.Name == "" || req.Half == "" || req.StartDate == "" || req.EndDate == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "team_id, name, half, start_date, end_date は必須です")
	}

	tid, err := team.NewTeamId(req.TeamID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	half, err := period.NewHalf(req.Half)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	dateRange, err := period.NewDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	pid, err := period.NewPeriodId(newUUID())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	now := time.Now().UTC()
	p, err := period.NewPeriod(pid, tid, req.Name, half, dateRange, now, now)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.repo.Save(ctx, p); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, toPeriodResponse(p))
}

// Get handles GET /api/v1/periods/:id
func (h *PeriodHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()

	pid, err := period.NewPeriodId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	p, err := h.repo.FindByID(ctx, pid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if p == nil {
		return echo.NewHTTPError(http.StatusNotFound, "期間が見つかりません")
	}
	return c.JSON(http.StatusOK, toPeriodResponse(*p))
}

// Delete handles DELETE /api/v1/periods/:id
func (h *PeriodHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	pid, err := period.NewPeriodId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.repo.Remove(ctx, pid); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

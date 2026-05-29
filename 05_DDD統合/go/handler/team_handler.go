package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"

)

// TeamHandler handles /api/v1/teams endpoints.
type TeamHandler struct {
	teamRepo repository.TeamRepository
}

// NewTeamHandler creates a TeamHandler with the given repository.
func NewTeamHandler(teamRepo repository.TeamRepository) *TeamHandler {
	return &TeamHandler{teamRepo: teamRepo}
}

// ── リクエスト / レスポンス DTO ────────────────────────────────────────────────

type createTeamRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type addMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"` // "admin" | "member"
}

type teamMemberResponse struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

type teamResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description *string              `json:"description"`
	Members     []teamMemberResponse `json:"members"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

func toTeamResponse(t team.Team) teamResponse {
	members := make([]teamMemberResponse, len(t.Members()))
	for i, m := range t.Members() {
		members[i] = teamMemberResponse{
			ID:       m.ID().Value(),
			UserID:   m.UserID().String(),
			Role:     m.Role().Value(),
			JoinedAt: m.JoinedAt().Format(time.RFC3339),
		}
	}
	return teamResponse{
		ID:          t.ID().Value(),
		Name:        t.Name(),
		Description: t.Description(),
		Members:     members,
		CreatedAt:   t.CreatedAt().Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt().Format(time.RFC3339),
	}
}

// ── ハンドラー実装 ─────────────────────────────────────────────────────────────

// List handles GET /api/v1/teams
func (h *TeamHandler) List(c echo.Context) error {
	ctx := c.Request().Context()

	teams, err := h.teamRepo.FindAll(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	res := make([]teamResponse, len(teams))
	for i, t := range teams {
		res[i] = toTeamResponse(t)
	}
	return c.JSON(http.StatusOK, res)
}

// Create handles POST /api/v1/teams
func (h *TeamHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var req createTeamRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストのパースに失敗しました: "+err.Error())
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name は必須です")
	}

	tid, err := team.NewTeamId(newUUID())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	now := time.Now().UTC()
	t, err := team.NewTeam(tid, req.Name, req.Description, nil, now, now)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.teamRepo.Save(ctx, t); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, toTeamResponse(t))
}

// Get handles GET /api/v1/teams/:id
func (h *TeamHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()

	tid, err := team.NewTeamId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	t, err := h.teamRepo.FindByID(ctx, tid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if t == nil {
		return echo.NewHTTPError(http.StatusNotFound, "チームが見つかりません")
	}
	return c.JSON(http.StatusOK, toTeamResponse(*t))
}

// Delete handles DELETE /api/v1/teams/:id
func (h *TeamHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	tid, err := team.NewTeamId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.teamRepo.Remove(ctx, tid); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// AddMember handles POST /api/v1/teams/:id/members
func (h *TeamHandler) AddMember(c echo.Context) error {
	ctx := c.Request().Context()

	tid, err := team.NewTeamId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var req addMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストのパースに失敗しました: "+err.Error())
	}
	if req.UserID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id は必須です")
	}
	roleStr := req.Role
	if roleStr == "" {
		roleStr = "member"
	}

	uid, err := user.NewUserId(req.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	role, err := team.NewRole(roleStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Read-Modify-Write パターン
	t, err := h.teamRepo.FindByID(ctx, tid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if t == nil {
		return echo.NewHTTPError(http.StatusNotFound, "チームが見つかりません")
	}

	memberId, err := team.NewTeamMemberId(newUUID())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	member, err := team.NewTeamMember(memberId, tid, uid, role, time.Now().UTC())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := t.AddMember(member); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}

	if err := h.teamRepo.Save(ctx, *t); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, toTeamResponse(*t))
}

// RemoveMember handles DELETE /api/v1/teams/:id/members/:member_id
// :member_id はここでは user_id として扱う（シンプル化のため）
func (h *TeamHandler) RemoveMember(c echo.Context) error {
	ctx := c.Request().Context()

	tid, err := team.NewTeamId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	uid, err := user.NewUserId(c.Param("user_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Read-Modify-Write パターン
	t, err := h.teamRepo.FindByID(ctx, tid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if t == nil {
		return echo.NewHTTPError(http.StatusNotFound, "チームが見つかりません")
	}

	t.RemoveMember(uid)

	if err := h.teamRepo.Save(ctx, *t); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// Package handler implements HTTP handlers for the OKR API.
package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

// UserHandler handles /api/v1/users endpoints.
type UserHandler struct {
	repo repository.UserRepository
}

// NewUserHandler creates a UserHandler with the given repository.
func NewUserHandler(repo repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// ── リクエスト / レスポンス DTO ────────────────────────────────────────────────

type createUserRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

type userResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func toUserResponse(u user.User) userResponse {
	return userResponse{
		ID:           u.ID().Value(),
		Name:         u.Name(),
		Email:        u.Email().Value(),
		PasswordHash: u.PasswordHash(),
		CreatedAt:    u.CreatedAt().Format(time.RFC3339),
		UpdatedAt:    u.UpdatedAt().Format(time.RFC3339),
	}
}

// ── ハンドラー実装 ─────────────────────────────────────────────────────────────

// List handles GET /api/v1/users
func (h *UserHandler) List(c echo.Context) error {
	ctx := c.Request().Context()

	users, err := h.repo.FindAll(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	res := make([]userResponse, len(users))
	for i, u := range users {
		res[i] = toUserResponse(u)
	}
	return c.JSON(http.StatusOK, res)
}

// Create handles POST /api/v1/users
func (h *UserHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var req createUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストのパースに失敗しました: "+err.Error())
	}
	if req.Name == "" || req.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name と email は必須です")
	}

	// メールアドレス重複チェック
	email, err := user.NewEmail(req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	existing, err := h.repo.FindByEmail(ctx, email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if existing != nil {
		return echo.NewHTTPError(http.StatusConflict, "そのメールアドレスは既に使用されています")
	}

	// UUID 生成
	uid, err := user.NewUserId(newUUID())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	now := time.Now().UTC()
	u, err := user.NewUser(uid, req.Name, email, req.PasswordHash, now, now)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.repo.Save(ctx, u); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, toUserResponse(u))
}

// Get handles GET /api/v1/users/:id
func (h *UserHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()

	uid, err := user.NewUserId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	u, err := h.repo.FindByID(ctx, uid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if u == nil {
		return echo.NewHTTPError(http.StatusNotFound, "ユーザーが見つかりません")
	}
	return c.JSON(http.StatusOK, toUserResponse(*u))
}

// Delete handles DELETE /api/v1/users/:id
func (h *UserHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	uid, err := user.NewUserId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.repo.Remove(ctx, uid); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

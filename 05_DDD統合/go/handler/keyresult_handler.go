package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/keyresult"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

// KeyResultHandler handles /api/v1/key_results endpoints.
type KeyResultHandler struct {
	repo repository.KeyResultRepository
}

// NewKeyResultHandler creates a KeyResultHandler with the given repository.
func NewKeyResultHandler(repo repository.KeyResultRepository) *KeyResultHandler {
	return &KeyResultHandler{repo: repo}
}

// ── リクエスト / レスポンス DTO ────────────────────────────────────────────────

type createKeyResultRequest struct {
	ObjectiveID  string   `json:"objective_id"`
	OwnerID      string   `json:"owner_id"`
	Title        string   `json:"title"`
	KrType       string   `json:"kr_type"` // "numeric" or "checkbox"
	TargetValue  *float64 `json:"target_value"`
	DisplayOrder int      `json:"display_order"`
}

type keyResultResponse struct {
	ID           string   `json:"id"`
	ObjectiveID  string   `json:"objective_id"`
	OwnerID      string   `json:"owner_id"`
	Title        string   `json:"title"`
	KrType       string   `json:"kr_type"`
	TargetValue  *float64 `json:"target_value"`
	CurrentValue *float64 `json:"current_value"`
	IsCompleted  bool     `json:"is_completed"`
	DisplayOrder int      `json:"display_order"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

func toKeyResultResponse(kr keyresult.KeyResult) keyResultResponse {
	return keyResultResponse{
		ID:           kr.ID().Value(),
		ObjectiveID:  kr.ObjectiveID().Value(),
		OwnerID:      kr.OwnerID().Value(),
		Title:        kr.Title(),
		KrType:       string(kr.KrType()),
		TargetValue:  kr.TargetValue(),
		CurrentValue: kr.CurrentValue(),
		IsCompleted:  kr.IsCompleted(),
		DisplayOrder: kr.DisplayOrder(),
		CreatedAt:    kr.CreatedAt().Format(time.RFC3339),
		UpdatedAt:    kr.UpdatedAt().Format(time.RFC3339),
	}
}

// ── ハンドラー実装 ─────────────────────────────────────────────────────────────

// List handles GET /api/v1/key_results
// クエリパラメータ: objective_id または owner_id（どちらか必須）
func (h *KeyResultHandler) List(c echo.Context) error {
	ctx := c.Request().Context()

	if objectiveIDStr := c.QueryParam("objective_id"); objectiveIDStr != "" {
		oid, err := objective.NewObjectiveId(objectiveIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		krs, err := h.repo.FindByObjectiveID(ctx, oid)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		res := make([]keyResultResponse, len(krs))
		for i, kr := range krs {
			res[i] = toKeyResultResponse(kr)
		}
		return c.JSON(http.StatusOK, res)
	}

	if ownerIDStr := c.QueryParam("owner_id"); ownerIDStr != "" {
		uid, err := user.NewUserId(ownerIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		krs, err := h.repo.FindByOwnerID(ctx, uid)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		res := make([]keyResultResponse, len(krs))
		for i, kr := range krs {
			res[i] = toKeyResultResponse(kr)
		}
		return c.JSON(http.StatusOK, res)
	}

	return echo.NewHTTPError(http.StatusBadRequest, "objective_id または owner_id クエリパラメータを指定してください")
}

// Create handles POST /api/v1/key_results
func (h *KeyResultHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var req createKeyResultRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "リクエストのパースに失敗しました: "+err.Error())
	}
	if req.ObjectiveID == "" || req.OwnerID == "" || req.Title == "" || req.KrType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "objective_id, owner_id, title, kr_type は必須です")
	}

	oid, err := objective.NewObjectiveId(req.ObjectiveID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	uid, err := user.NewUserId(req.OwnerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	kid, err := keyresult.NewKeyResultId(newUUID())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	now := time.Now().UTC()

	var kr keyresult.KeyResult
	switch req.KrType {
	case string(keyresult.KrTypeNumeric):
		if req.TargetValue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "numeric KR には target_value が必須です")
		}
		kr, err = keyresult.NewNumericKeyResult(kid, oid, uid, req.Title, *req.TargetValue, req.DisplayOrder, now, now)
	case string(keyresult.KrTypeCheckbox):
		kr, err = keyresult.NewCheckboxKeyResult(kid, oid, uid, req.Title, req.DisplayOrder, now, now)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "kr_type は 'numeric' または 'checkbox' を指定してください")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.repo.Save(ctx, kr); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, toKeyResultResponse(kr))
}

// Get handles GET /api/v1/key_results/:id
func (h *KeyResultHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()

	kid, err := keyresult.NewKeyResultId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	kr, err := h.repo.FindByID(ctx, kid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if kr == nil {
		return echo.NewHTTPError(http.StatusNotFound, "キーリザルトが見つかりません")
	}
	return c.JSON(http.StatusOK, toKeyResultResponse(*kr))
}

// Delete handles DELETE /api/v1/key_results/:id
func (h *KeyResultHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	kid, err := keyresult.NewKeyResultId(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.repo.Remove(ctx, kid); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

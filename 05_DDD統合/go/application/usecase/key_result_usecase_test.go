package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/application/usecase"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/application/usecase/mock"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/keyresult"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

const (
	testObjUUID = "00000000-0000-4000-8000-000000000020"
	testKrUUID  = "00000000-0000-4000-8000-000000000030"
	testOwnerUUID = "00000000-0000-4000-8000-000000000040"
	testLogUUID = "00000000-0000-4000-8000-000000000050"
)

// seedNumericKR inserts a numeric KR into the mock repo.
func seedNumericKR(t *testing.T, repo *mock.KeyResultRepository) keyresult.KeyResult {
	t.Helper()

	kid, err := keyresult.NewKeyResultId(testKrUUID)
	require.NoError(t, err)
	objID, err := objective.NewObjectiveId(testObjUUID)
	require.NoError(t, err)
	ownerID, err := user.NewUserId(testOwnerUUID)
	require.NoError(t, err)

	now := time.Now().UTC()
	kr, err := keyresult.NewNumericKeyResult(kid, objID, ownerID, "売上100万円達成", 1_000_000, 1, now, now)
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), kr))
	return kr
}

// seedCheckboxKR inserts a checkbox KR into the mock repo.
func seedCheckboxKR(t *testing.T, repo *mock.KeyResultRepository) keyresult.KeyResult {
	t.Helper()

	kid, err := keyresult.NewKeyResultId(testKrUUID)
	require.NoError(t, err)
	objID, err := objective.NewObjectiveId(testObjUUID)
	require.NoError(t, err)
	ownerID, err := user.NewUserId(testOwnerUUID)
	require.NoError(t, err)

	now := time.Now().UTC()
	kr, err := keyresult.NewCheckboxKeyResult(kid, objID, ownerID, "ドキュメント作成完了", 1, now, now)
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), kr))
	return kr
}

// ─── CreateNumericKeyResult ──────────────────────────────────────────────────

func TestKRUseCase_CreateNumericKeyResult_正常系(t *testing.T) {
	repo := mock.NewKeyResultRepository()
	uc := usecase.NewKeyResultUseCase(repo)

	kr, err := uc.CreateNumericKeyResult(context.Background(), usecase.CreateNumericKRInput{
		ObjectiveID:  testObjUUID,
		OwnerID:      testOwnerUUID,
		Title:        "売上目標",
		TargetValue:  500_000,
		DisplayOrder: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, kr)
	assert.Equal(t, keyresult.KrTypeNumeric, kr.KrType())
	assert.Equal(t, "売上目標", kr.Title())
	assert.NotNil(t, kr.TargetValue())
	assert.Equal(t, 500_000.0, *kr.TargetValue())
}

func TestKRUseCase_CreateNumericKeyResult_空タイトルエラー(t *testing.T) {
	repo := mock.NewKeyResultRepository()
	uc := usecase.NewKeyResultUseCase(repo)

	_, err := uc.CreateNumericKeyResult(context.Background(), usecase.CreateNumericKRInput{
		ObjectiveID: testObjUUID,
		OwnerID:     testOwnerUUID,
		Title:       "",
	})

	require.Error(t, err)
}

// ─── CreateCheckboxKeyResult ─────────────────────────────────────────────────

func TestKRUseCase_CreateCheckboxKeyResult_正常系(t *testing.T) {
	repo := mock.NewKeyResultRepository()
	uc := usecase.NewKeyResultUseCase(repo)

	kr, err := uc.CreateCheckboxKeyResult(context.Background(), usecase.CreateCheckboxKRInput{
		ObjectiveID:  testObjUUID,
		OwnerID:      testOwnerUUID,
		Title:        "書類提出",
		DisplayOrder: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, kr)
	assert.Equal(t, keyresult.KrTypeCheckbox, kr.KrType())
	assert.False(t, kr.IsCompleted())
}

// ─── UpdateNumericProgress ───────────────────────────────────────────────────

func TestKRUseCase_UpdateNumericProgress_正常系(t *testing.T) {
	repo := mock.NewKeyResultRepository()
	uc := usecase.NewKeyResultUseCase(repo)

	seedNumericKR(t, repo)

	updated, err := uc.UpdateNumericProgress(context.Background(), usecase.UpdateNumericProgressInput{
		KeyResultID: testKrUUID,
		RecordedBy:  testOwnerUUID,
		Value:       300_000,
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.NotNil(t, updated.CurrentValue())
	assert.Equal(t, 300_000.0, *updated.CurrentValue())
	assert.Len(t, updated.ProgressLogs(), 1)
}

func TestKRUseCase_UpdateNumericProgress_KR存在しないエラー(t *testing.T) {
	repo := mock.NewKeyResultRepository()
	uc := usecase.NewKeyResultUseCase(repo)

	_, err := uc.UpdateNumericProgress(context.Background(), usecase.UpdateNumericProgressInput{
		KeyResultID: testKrUUID,
		RecordedBy:  testOwnerUUID,
		Value:       100,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "KRが見つかりません")
}

// ─── 型ミスマッチエラー（checkbox KR に numeric を渡す）────────────────────────

func TestKRUseCase_UpdateNumericProgress_checkboxKRへの型ミスマッチエラー(t *testing.T) {
	repo := mock.NewKeyResultRepository()
	uc := usecase.NewKeyResultUseCase(repo)

	// checkbox KR を作成
	seedCheckboxKR(t, repo)

	// numeric の進捗更新を試みる → エラー
	_, err := uc.UpdateNumericProgress(context.Background(), usecase.UpdateNumericProgressInput{
		KeyResultID: testKrUUID,
		RecordedBy:  testOwnerUUID,
		Value:       100,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "numeric KR ではありません")
}

// ─── UpdateCheckboxProgress ──────────────────────────────────────────────────

func TestKRUseCase_UpdateCheckboxProgress_正常系(t *testing.T) {
	repo := mock.NewKeyResultRepository()
	uc := usecase.NewKeyResultUseCase(repo)

	seedCheckboxKR(t, repo)

	updated, err := uc.UpdateCheckboxProgress(context.Background(), usecase.UpdateCheckboxProgressInput{
		KeyResultID: testKrUUID,
		RecordedBy:  testOwnerUUID,
		Completed:   true,
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.True(t, updated.IsCompleted())
	assert.Len(t, updated.ProgressLogs(), 1)
}

func TestKRUseCase_UpdateCheckboxProgress_numericKRへの型ミスマッチエラー(t *testing.T) {
	repo := mock.NewKeyResultRepository()
	uc := usecase.NewKeyResultUseCase(repo)

	// numeric KR を作成
	seedNumericKR(t, repo)

	// checkbox の進捗更新を試みる → エラー
	_, err := uc.UpdateCheckboxProgress(context.Background(), usecase.UpdateCheckboxProgressInput{
		KeyResultID: testKrUUID,
		RecordedBy:  testOwnerUUID,
		Completed:   true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checkbox KR ではありません")
}

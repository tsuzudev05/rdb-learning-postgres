package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/application/usecase"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/application/usecase/mock"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// ─── テスト用ヘルパー ─────────────────────────────────────────────────────────

const (
	testUUID1 = "00000000-0000-4000-8000-000000000001"
	testUUID2 = "00000000-0000-4000-8000-000000000002"
)

func mustUserId(t *testing.T, v string) user.UserId {
	t.Helper()
	id, err := user.NewUserId(v)
	require.NoError(t, err)
	return id
}

func mustEmail(t *testing.T, v string) user.Email {
	t.Helper()
	email, err := user.NewEmail(v)
	require.NoError(t, err)
	return email
}

func newTestUser(t *testing.T, id, email string) user.User {
	t.Helper()
	u, err := user.NewUser(mustUserId(t, id), "テストユーザー", mustEmail(t, email), "hash", time.Now(), time.Now())
	require.NoError(t, err)
	return u
}

// ─── CreateUser ───────────────────────────────────────────────────────────────

func TestUserUseCase_CreateUser_正常系(t *testing.T) {
	repo := mock.NewUserRepository()
	uc := usecase.NewUserUseCase(repo)

	u, err := uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Name:  "田中太郎",
		Email: "tanaka@example.com",
	})

	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "田中太郎", u.Name())
	assert.Equal(t, "tanaka@example.com", u.Email().Value())
	// リポジトリに保存されていること
	assert.Len(t, repo.Users, 1)
}

func TestUserUseCase_CreateUser_重複メールエラー(t *testing.T) {
	repo := mock.NewUserRepository()
	uc := usecase.NewUserUseCase(repo)

	// 先にユーザーを登録
	existing := newTestUser(t, testUUID1, "dup@example.com")
	require.NoError(t, repo.Save(context.Background(), existing))

	// 同じメールで作成しようとするとエラー
	_, err := uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Name:  "別のユーザー",
		Email: "dup@example.com",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "既に使用されています")
	// 新規保存されていないこと（元の1件のまま）
	assert.Len(t, repo.Users, 1)
}

func TestUserUseCase_CreateUser_無効なメール(t *testing.T) {
	repo := mock.NewUserRepository()
	uc := usecase.NewUserUseCase(repo)

	_, err := uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Name:  "テスト",
		Email: "invalid-email",
	})

	require.Error(t, err)
	assert.Len(t, repo.Users, 0)
}

func TestUserUseCase_CreateUser_リポジトリエラー(t *testing.T) {
	repo := mock.NewUserRepository()
	repo.ErrSave = mock.ErrDB
	uc := usecase.NewUserUseCase(repo)

	_, err := uc.CreateUser(context.Background(), usecase.CreateUserInput{
		Name:  "テスト",
		Email: "test@example.com",
	})

	require.Error(t, err)
}

// ─── GetUser ─────────────────────────────────────────────────────────────────

func TestUserUseCase_GetUser_存在する(t *testing.T) {
	repo := mock.NewUserRepository()
	existing := newTestUser(t, testUUID1, "get@example.com")
	require.NoError(t, repo.Save(context.Background(), existing))

	uc := usecase.NewUserUseCase(repo)
	u, err := uc.GetUser(context.Background(), mustUserId(t, testUUID1))

	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, testUUID1, u.ID().Value())
}

func TestUserUseCase_GetUser_存在しない(t *testing.T) {
	repo := mock.NewUserRepository()
	uc := usecase.NewUserUseCase(repo)

	u, err := uc.GetUser(context.Background(), mustUserId(t, testUUID1))

	require.NoError(t, err)
	assert.Nil(t, u)
}

// ─── DeleteUser ──────────────────────────────────────────────────────────────

func TestUserUseCase_DeleteUser_冪等性(t *testing.T) {
	repo := mock.NewUserRepository()
	existing := newTestUser(t, testUUID1, "del@example.com")
	require.NoError(t, repo.Save(context.Background(), existing))

	uc := usecase.NewUserUseCase(repo)

	// 1回目：削除成功
	err := uc.DeleteUser(context.Background(), mustUserId(t, testUUID1))
	require.NoError(t, err)
	assert.Len(t, repo.Users, 0)

	// 2回目：存在しなくてもエラーなし（冪等）
	err = uc.DeleteUser(context.Background(), mustUserId(t, testUUID1))
	require.NoError(t, err)
}

// ─── ListUsers ───────────────────────────────────────────────────────────────

func TestUserUseCase_ListUsers(t *testing.T) {
	repo := mock.NewUserRepository()
	require.NoError(t, repo.Save(context.Background(), newTestUser(t, testUUID1, "a@example.com")))
	require.NoError(t, repo.Save(context.Background(), newTestUser(t, testUUID2, "b@example.com")))

	uc := usecase.NewUserUseCase(repo)
	users, err := uc.ListUsers(context.Background())

	require.NoError(t, err)
	assert.Len(t, users, 2)
}

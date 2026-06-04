package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/application/usecase"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/application/usecase/mock"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

const (
	testTeamUUID   = "00000000-0000-4000-8000-000000000010"
	testMemberUUID = "00000000-0000-4000-8000-000000000011"
	testMemberUUID2 = "00000000-0000-4000-8000-000000000012"
)

func mustTeamId(t *testing.T, v string) team.TeamId {
	t.Helper()
	id, err := team.NewTeamId(v)
	require.NoError(t, err)
	return id
}

func mustTeamMemberId(t *testing.T, v string) team.TeamMemberId {
	t.Helper()
	id, err := team.NewTeamMemberId(v)
	require.NoError(t, err)
	return id
}

// seedTeamWithMember puts a team with one member into the repo and returns both.
func seedTeamWithMember(t *testing.T, repo *mock.TeamRepository) (team.Team, team.TeamMember) {
	t.Helper()

	tid := mustTeamId(t, testTeamUUID)
	uid, err := user.NewUserId(testMemberUUID)
	require.NoError(t, err)

	mid := mustTeamMemberId(t, "00000000-0000-4000-8000-000000000099")
	member, err := team.NewTeamMember(mid, tid, uid, team.Member(), time.Now())
	require.NoError(t, err)

	tm, err := team.NewTeam(tid, "テストチーム", nil, []team.TeamMember{member}, time.Now(), time.Now())
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), tm))
	return tm, member
}

// ─── CreateTeam ──────────────────────────────────────────────────────────────

func TestTeamUseCase_CreateTeam_正常系(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	desc := "説明テキスト"
	tm, err := uc.CreateTeam(context.Background(), usecase.CreateTeamInput{
		Name:        "開発チーム",
		Description: &desc,
	})

	require.NoError(t, err)
	require.NotNil(t, tm)
	assert.Equal(t, "開発チーム", tm.Name())
	assert.Equal(t, &desc, tm.Description())
	assert.Len(t, repo.Teams, 1)
}

func TestTeamUseCase_CreateTeam_空チーム名エラー(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	_, err := uc.CreateTeam(context.Background(), usecase.CreateTeamInput{Name: ""})

	require.Error(t, err)
	assert.Len(t, repo.Teams, 0)
}

// ─── AddMember ───────────────────────────────────────────────────────────────

func TestTeamUseCase_AddMember_正常系(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	// チーム（メンバーなし）を作成
	tid := mustTeamId(t, testTeamUUID)
	tm, err := team.NewTeam(tid, "チーム", nil, nil, time.Now(), time.Now())
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), tm))

	// メンバー追加
	result, err := uc.AddMember(context.Background(), usecase.AddMemberInput{
		TeamID: testTeamUUID,
		UserID: testMemberUUID,
		Role:   "member",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Members(), 1)
	assert.Equal(t, testMemberUUID, result.Members()[0].UserID().String())
}

func TestTeamUseCase_AddMember_重複メンバーエラー(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	seedTeamWithMember(t, repo)

	// 同じ user を再度追加 → エラー
	_, err := uc.AddMember(context.Background(), usecase.AddMemberInput{
		TeamID: testTeamUUID,
		UserID: testMemberUUID,
		Role:   "member",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "既にメンバーです")
}

func TestTeamUseCase_AddMember_チームなしエラー(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	_, err := uc.AddMember(context.Background(), usecase.AddMemberInput{
		TeamID: testTeamUUID,
		UserID: testMemberUUID,
		Role:   "member",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "チームが見つかりません")
}

// ─── RemoveMember ────────────────────────────────────────────────────────────

func TestTeamUseCase_RemoveMember_正常系(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	seedTeamWithMember(t, repo)

	result, err := uc.RemoveMember(context.Background(), usecase.RemoveMemberInput{
		TeamID: testTeamUUID,
		UserID: testMemberUUID,
	})

	require.NoError(t, err)
	assert.Len(t, result.Members(), 0)
}

func TestTeamUseCase_RemoveMember_存在しないメンバーはNoOp(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	seedTeamWithMember(t, repo)

	// 存在しない user を削除してもエラーなし（冪等）
	result, err := uc.RemoveMember(context.Background(), usecase.RemoveMemberInput{
		TeamID: testTeamUUID,
		UserID: testMemberUUID2,
	})

	require.NoError(t, err)
	// 既存メンバーは残る
	assert.Len(t, result.Members(), 1)
}

// ─── ChangeMemberRole ────────────────────────────────────────────────────────

func TestTeamUseCase_ChangeMemberRole_memberからadminへ(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	seedTeamWithMember(t, repo)

	result, err := uc.ChangeMemberRole(context.Background(), usecase.ChangeMemberRoleInput{
		TeamID: testTeamUUID,
		UserID: testMemberUUID,
		Role:   "admin",
	})

	require.NoError(t, err)
	require.Len(t, result.Members(), 1)
	assert.Equal(t, "admin", result.Members()[0].Role().Value())
}

func TestTeamUseCase_ChangeMemberRole_非メンバーへの変更はエラー(t *testing.T) {
	repo := mock.NewTeamRepository()
	uc := usecase.NewTeamUseCase(repo)

	seedTeamWithMember(t, repo)

	_, err := uc.ChangeMemberRole(context.Background(), usecase.ChangeMemberRoleInput{
		TeamID: testTeamUUID,
		UserID: testMemberUUID2, // 存在しないメンバー
		Role:   "admin",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "メンバーではありません")
}

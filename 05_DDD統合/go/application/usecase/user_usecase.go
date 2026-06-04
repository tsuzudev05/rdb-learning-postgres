// Package usecase implements application use cases for the OKR domain.
// Use cases orchestrate domain objects and repositories; they contain no
// HTTP / DB / framework dependencies.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

// UserUseCase handles application logic for the User aggregate.
type UserUseCase struct {
	repo repository.UserRepository
}

// NewUserUseCase creates a UserUseCase with the given repository.
func NewUserUseCase(repo repository.UserRepository) *UserUseCase {
	return &UserUseCase{repo: repo}
}

// CreateUserInput holds the input data for creating a user.
type CreateUserInput struct {
	Name         string
	Email        string
	PasswordHash string
}

// CreateUser validates input, checks for duplicate email, then persists the user.
// Returns an error if the email is already in use.
func (uc *UserUseCase) CreateUser(ctx context.Context, in CreateUserInput) (*user.User, error) {
	email, err := user.NewEmail(in.Email)
	if err != nil {
		return nil, err
	}

	existing, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase.CreateUser: メール検索エラー: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("UserUseCase.CreateUser: メールアドレス %s は既に使用されています", in.Email)
	}

	uid, err := user.NewUserId(uuid.New().String())
	if err != nil {
		return nil, fmt.Errorf("UserUseCase.CreateUser: UUID生成エラー: %w", err)
	}

	now := time.Now().UTC()
	u, err := user.NewUser(uid, in.Name, email, in.PasswordHash, now, now)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("UserUseCase.CreateUser: 保存エラー: %w", err)
	}
	return &u, nil
}

// GetUser returns the user with the given ID, or (nil, nil) if not found.
func (uc *UserUseCase) GetUser(ctx context.Context, id user.UserId) (*user.User, error) {
	u, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase.GetUser: %w", err)
	}
	return u, nil
}

// ListUsers returns all users.
func (uc *UserUseCase) ListUsers(ctx context.Context) ([]user.User, error) {
	users, err := uc.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase.ListUsers: %w", err)
	}
	return users, nil
}

// DeleteUser removes the user by ID. No-op if the user does not exist (冪等).
func (uc *UserUseCase) DeleteUser(ctx context.Context, id user.UserId) error {
	if err := uc.repo.Remove(ctx, id); err != nil {
		return fmt.Errorf("UserUseCase.DeleteUser: %w", err)
	}
	return nil
}

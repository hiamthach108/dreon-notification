package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/pkg/validator"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IUserFCMTokenSvc interface {
	Register(ctx context.Context, req *aggregate.RegisterUserFCMTokenReq) (*aggregate.UserFCMTokenAggregate, error)
	ListForUser(ctx context.Context, userID string) ([]aggregate.UserFCMTokenAggregate, error)
	DeleteForUser(ctx context.Context, id, userID string) error
}

type UserFCMTokenSvc struct {
	repo repository.IUserFCMTokenRepository
}

func NewUserFCMTokenSvc(repo repository.IUserFCMTokenRepository) IUserFCMTokenSvc {
	return &UserFCMTokenSvc{repo: repo}
}

func (s *UserFCMTokenSvc) Register(ctx context.Context, req *aggregate.RegisterUserFCMTokenReq) (*aggregate.UserFCMTokenAggregate, error) {
	if err := validator.ValidateStruct(req); err != nil {
		return nil, errorx.Wrap(errorx.ErrBadRequest, validator.FormatValidationError(err))
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return nil, errorx.New(errorx.ErrBadRequest, "token is required")
	}

	var meta datatypes.JSON
	if len(req.DeviceMetadata) > 0 {
		b, err := json.Marshal(req.DeviceMetadata)
		if err != nil {
			return nil, errorx.Wrap(errorx.ErrBadRequest, fmt.Errorf("deviceMetadata: %w", err))
		}
		meta = b
	}

	existing := s.repo.FindByToken(ctx, token)
	if existing != nil {
		updates := model.UserFCMToken{
			Platform:       req.Platform,
			DeviceMetadata: meta,
		}
		updates.CreatedBy = req.UserID
		if err := s.repo.Update(ctx, existing.ID, updates, "CreatedBy", "Platform", "DeviceMetadata"); err != nil {
			return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("update fcm token: %w", err))
		}
		updated := s.repo.FindOneById(ctx, existing.ID)
		if updated == nil {
			return nil, errorx.New(errorx.ErrInternal, "fcm token updated but row not found")
		}
		var agg aggregate.UserFCMTokenAggregate
		agg.FromModel(updated)
		return &agg, nil
	}

	row := &model.UserFCMToken{
		Token:          token,
		Platform:       req.Platform,
		DeviceMetadata: meta,
	}
	row.CreatedBy = req.UserID

	created, err := s.repo.Create(ctx, row)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("create fcm token: %w", err))
	}
	if created == nil || created.ID == "" {
		return nil, errorx.New(errorx.ErrInternal, "fcm token created but ID is empty")
	}
	var agg aggregate.UserFCMTokenAggregate
	agg.FromModel(created)
	return &agg, nil
}

func (s *UserFCMTokenSvc) ListForUser(ctx context.Context, userID string) ([]aggregate.UserFCMTokenAggregate, error) {
	if userID == "" {
		return nil, errorx.New(errorx.ErrBadRequest, "user id is required")
	}
	rows, err := s.repo.ListByCreatedBy(ctx, userID)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("list fcm tokens: %w", err))
	}
	out := make([]aggregate.UserFCMTokenAggregate, 0, len(rows))
	for i := range rows {
		var agg aggregate.UserFCMTokenAggregate
		agg.FromModel(&rows[i])
		out = append(out, agg)
	}
	return out, nil
}

func (s *UserFCMTokenSvc) DeleteForUser(ctx context.Context, id, userID string) error {
	if id == "" || userID == "" {
		return errorx.New(errorx.ErrBadRequest, "id and user id are required")
	}
	err := s.repo.DeleteByIdAndCreatedBy(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.New(errorx.ErrNotFound, "fcm token not found")
		}
		return errorx.Wrap(errorx.ErrInternal, fmt.Errorf("delete fcm token: %w", err))
	}
	return nil
}

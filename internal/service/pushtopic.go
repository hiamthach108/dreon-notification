package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/pkg/validator"
)

type IPushTopicSvc interface {
	GetAll(ctx context.Context) ([]aggregate.PushTopicAggregate, error)
	Create(ctx context.Context, req *aggregate.CreatePushTopicReq) (*aggregate.PushTopicAggregate, error)
	Update(ctx context.Context, id string, req *aggregate.UpdatePushTopicReq) error
}

type PushTopicSvc struct {
	repo repository.IPushTopicRepository
}

// NewPushTopicSvc registers and resolves FCM topic metadata (name, description, active flag).
func NewPushTopicSvc(repo repository.IPushTopicRepository) IPushTopicSvc {
	return &PushTopicSvc{repo: repo}
}

func (s *PushTopicSvc) GetAll(ctx context.Context) ([]aggregate.PushTopicAggregate, error) {
	rows, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("list push topics: %w", err))
	}
	out := make([]aggregate.PushTopicAggregate, 0, len(rows))
	for i := range rows {
		var agg aggregate.PushTopicAggregate
		agg.FromModel(&rows[i])
		out = append(out, agg)
	}
	return out, nil
}

func (s *PushTopicSvc) Create(ctx context.Context, req *aggregate.CreatePushTopicReq) (*aggregate.PushTopicAggregate, error) {
	if err := validator.ValidateStruct(req); err != nil {
		return nil, errorx.Wrap(errorx.ErrBadRequest, validator.FormatValidationError(err))
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorx.New(errorx.ErrBadRequest, "name is required")
	}
	if existing := s.repo.FindByName(ctx, name); existing != nil {
		return nil, errorx.New(errorx.ErrConflict, "push topic name already exists")
	}

	topic := &model.PushTopic{
		Name:        name,
		Description: req.Description,
		IsActive:    true,
	}
	if req.IsActive != nil {
		topic.IsActive = *req.IsActive
	}

	created, err := s.repo.Create(ctx, topic)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("create push topic: %w", err))
	}
	if created == nil || created.ID == "" {
		return nil, errorx.New(errorx.ErrInternal, "push topic created but ID is empty")
	}
	var agg aggregate.PushTopicAggregate
	agg.FromModel(created)
	return &agg, nil
}

func (s *PushTopicSvc) Update(ctx context.Context, id string, req *aggregate.UpdatePushTopicReq) error {
	if strings.TrimSpace(id) == "" {
		return errorx.New(errorx.ErrBadRequest, "id is required")
	}
	if err := validator.ValidateStruct(req); err != nil {
		return errorx.Wrap(errorx.ErrBadRequest, validator.FormatValidationError(err))
	}
	current := s.repo.FindOneById(ctx, id)
	if current == nil {
		return errorx.New(errorx.ErrNotFound, "push topic not found")
	}

	updates := model.PushTopic{}
	var fields []string

	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName == "" {
			return errorx.New(errorx.ErrBadRequest, "name cannot be empty")
		}
		if other := s.repo.FindByName(ctx, newName); other != nil && other.ID != id {
			return errorx.New(errorx.ErrConflict, "push topic name already exists")
		}
		updates.Name = newName
		fields = append(fields, "Name")
	}
	if req.Description != nil {
		updates.Description = *req.Description
		fields = append(fields, "Description")
	}
	if req.IsActive != nil {
		updates.IsActive = *req.IsActive
		fields = append(fields, "IsActive")
	}
	if len(fields) == 0 {
		return errorx.New(errorx.ErrBadRequest, "no fields to update")
	}

	if err := s.repo.Update(ctx, id, updates, fields...); err != nil {
		return errorx.Wrap(errorx.ErrInternal, fmt.Errorf("update push topic: %w", err))
	}
	return nil
}

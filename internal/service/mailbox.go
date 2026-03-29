package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/hiamthach108/dreon-notification/internal/repository"
	"github.com/hiamthach108/dreon-notification/pkg/validator"
	"gorm.io/gorm"
)

type IMailboxSvc interface {
	Create(ctx context.Context, req *aggregate.CreateMailboxReq) (*aggregate.MailboxAggregate, error)
	ListForUser(ctx context.Context, userID string, limit int) ([]aggregate.MailboxAggregate, error)
	GetForUser(ctx context.Context, id, userID string) (*aggregate.MailboxAggregate, error)
	MarkAsRead(ctx context.Context, id, userID string) error
}

type MailboxSvc struct {
	repo repository.IMailboxRepository
}

func NewMailboxSvc(repo repository.IMailboxRepository) IMailboxSvc {
	return &MailboxSvc{repo: repo}
}

func (s *MailboxSvc) Create(ctx context.Context, req *aggregate.CreateMailboxReq) (*aggregate.MailboxAggregate, error) {
	if err := validator.ValidateStruct(req); err != nil {
		return nil, errorx.Wrap(errorx.ErrBadRequest, validator.FormatValidationError(err))
	}
	mailbox := req.ToModel()
	created, err := s.repo.Create(ctx, mailbox)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("create mailbox: %w", err))
	}
	if created == nil || created.ID == "" {
		return nil, errorx.New(errorx.ErrInternal, "mailbox created but ID is empty")
	}
	var agg aggregate.MailboxAggregate
	agg.FromModel(created)
	return &agg, nil
}

func (s *MailboxSvc) ListForUser(ctx context.Context, userID string, limit int) ([]aggregate.MailboxAggregate, error) {
	if userID == "" {
		return nil, errorx.New(errorx.ErrBadRequest, "user id is required")
	}
	rows, err := s.repo.ListByCreatedBy(ctx, userID, limit)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrInternal, fmt.Errorf("list mailbox: %w", err))
	}
	out := make([]aggregate.MailboxAggregate, 0, len(rows))
	for i := range rows {
		var agg aggregate.MailboxAggregate
		agg.FromModel(&rows[i])
		out = append(out, agg)
	}
	return out, nil
}

func (s *MailboxSvc) GetForUser(ctx context.Context, id, userID string) (*aggregate.MailboxAggregate, error) {
	if id == "" || userID == "" {
		return nil, errorx.New(errorx.ErrBadRequest, "id and user id are required")
	}
	row := s.repo.FindOneByIdAndCreatedBy(ctx, id, userID)
	if row == nil {
		return nil, errorx.New(errorx.ErrNotFound, "mailbox not found")
	}
	var agg aggregate.MailboxAggregate
	agg.FromModel(row)
	return &agg, nil
}

func (s *MailboxSvc) MarkAsRead(ctx context.Context, id, userID string) error {
	if id == "" || userID == "" {
		return errorx.New(errorx.ErrBadRequest, "id and user id are required")
	}
	err := s.repo.MarkRead(ctx, id, userID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.New(errorx.ErrNotFound, "mailbox not found")
		}
		return errorx.Wrap(errorx.ErrInternal, fmt.Errorf("mark mailbox read: %w", err))
	}
	return nil
}

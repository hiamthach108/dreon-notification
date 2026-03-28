package handler

import (
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/labstack/echo/v4"
)

type NotificationHandler struct {
	logger          logger.ILogger
	notificationSvc service.INotificationSvc
}

func NewNotificationHandler(logger logger.ILogger, notificationSvc service.INotificationSvc) *NotificationHandler {
	return &NotificationHandler{
		logger:          logger,
		notificationSvc: notificationSvc,
	}
}

func (h *NotificationHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.HandleSendNotification)
}

func (h *NotificationHandler) HandleSendNotification(c echo.Context) error {
	ctx := c.Request().Context()
	req, err := HandleValidateBind[aggregate.SendNotificationReq](c)
	if err != nil {
		return HandleError(c, err)
	}
	notificationID, err := h.notificationSvc.EnqueueNotification(ctx, &req)
	if err != nil {
		return HandleError(c, err)
	}
	return HandleSuccess(c, echo.Map{
		"notificationId": notificationID,
	})
}

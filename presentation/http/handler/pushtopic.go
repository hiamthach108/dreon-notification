package handler

import (
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/labstack/echo/v4"
)

type PushTopicHandler struct {
	logger       logger.ILogger
	pushTopicSvc service.IPushTopicSvc
}

func NewPushTopicHandler(logger logger.ILogger, pushTopicSvc service.IPushTopicSvc) *PushTopicHandler {
	return &PushTopicHandler{
		logger:       logger,
		pushTopicSvc: pushTopicSvc,
	}
}

func (h *PushTopicHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.HandleGetAllPushTopics)
	g.POST("", h.HandleCreatePushTopic)
	g.PUT("/:id", h.HandleUpdatePushTopic)
}

func (h *PushTopicHandler) HandleGetAllPushTopics(c echo.Context) error {
	ctx := c.Request().Context()
	pushTopics, err := h.pushTopicSvc.GetAll(ctx)
	if err != nil {
		return HandleError(c, err)
	}
	return HandleSuccess(c, pushTopics)
}

func (h *PushTopicHandler) HandleCreatePushTopic(c echo.Context) error {
	ctx := c.Request().Context()
	req, err := HandleValidateBind[aggregate.CreatePushTopicReq](c)
	if err != nil {
		return HandleError(c, err)
	}
	pushTopic, err := h.pushTopicSvc.Create(ctx, &req)
	if err != nil {
		return HandleError(c, err)
	}
	return HandleSuccess(c, pushTopic)
}

func (h *PushTopicHandler) HandleUpdatePushTopic(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	req, err := HandleValidateBind[aggregate.UpdatePushTopicReq](c)
	if err != nil {
		return HandleError(c, err)
	}
	err = h.pushTopicSvc.Update(ctx, id, &req)
	if err != nil {
		return HandleError(c, err)
	}
	return HandleSuccess(c, nil)
}

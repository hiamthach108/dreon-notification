package handler

import (
	"github.com/hiamthach108/dreon-notification/internal/aggregate"
	"github.com/hiamthach108/dreon-notification/internal/service"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/hiamthach108/dreon-notification/pkg/validator"
	"github.com/labstack/echo/v4"
)

type UserFCMTokenHandler struct {
	logger logger.ILogger
	svc    service.IUserFCMTokenSvc
}

func NewUserFCMTokenHandler(logger logger.ILogger, svc service.IUserFCMTokenSvc) *UserFCMTokenHandler {
	return &UserFCMTokenHandler{logger: logger, svc: svc}
}

func (h *UserFCMTokenHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.HandleRegister)
	g.GET("", h.HandleList)
	g.DELETE("/:id", h.HandleDelete)
}

func (h *UserFCMTokenHandler) HandleRegister(c echo.Context) error {
	ctx := c.Request().Context()
	req, err := HandleValidateBind[aggregate.RegisterUserFCMTokenReq](c)
	if err != nil {
		return HandleError(c, err)
	}
	out, err := h.svc.Register(ctx, &req)
	if err != nil {
		return HandleError(c, err)
	}
	return HandleSuccess(c, out)
}

func (h *UserFCMTokenHandler) HandleList(c echo.Context) error {
	ctx := c.Request().Context()
	var q aggregate.ListUserFCMTokenQuery
	if err := c.Bind(&q); err != nil {
		return HandleError(c, err)
	}
	if err := validator.ValidateStruct(&q); err != nil {
		return HandleError(c, err)
	}
	list, err := h.svc.ListForUser(ctx, q.UserID)
	if err != nil {
		return HandleError(c, err)
	}
	return HandleSuccess(c, list)
}

func (h *UserFCMTokenHandler) HandleDelete(c echo.Context) error {
	ctx := c.Request().Context()
	var q aggregate.DeleteUserFCMTokenQuery
	if err := c.Bind(&q); err != nil {
		return HandleError(c, err)
	}
	if err := validator.ValidateStruct(&q); err != nil {
		return HandleError(c, err)
	}
	id := c.Param("id")
	if err := h.svc.DeleteForUser(ctx, id, q.UserID); err != nil {
		return HandleError(c, err)
	}
	return HandleSuccess(c, nil)
}

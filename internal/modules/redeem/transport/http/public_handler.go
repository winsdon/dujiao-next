package redeemhttp

import (
	"errors"

	redeemapp "github.com/dujiao-next/internal/modules/redeem/application"
	redeemcontract "github.com/dujiao-next/internal/modules/redeem/contract"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

// PublicHandler 公开兑换接口。
type PublicHandler struct {
	svc *redeemapp.Service
}

func NewPublicHandler(svc *redeemapp.Service) *PublicHandler {
	if svc == nil {
		panic("redeem public handler: service is nil")
	}
	return &PublicHandler{svc: svc}
}

type codeRequest struct {
	Code string `json:"code" binding:"required"`
}

type submitRequest struct {
	Code       string `json:"code" binding:"required"`
	Contact    string `json:"contact" binding:"required"`
	Credential string `json:"credential" binding:"required"`
	Authorized bool   `json:"authorized"`
}

func (h *PublicHandler) Verify(c *gin.Context) {
	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	res, err := h.svc.Verify(c.Request.Context(), req.Code)
	if err != nil {
		respondRedeemError(c, err)
		return
	}
	response.Success(c, res)
}

func (h *PublicHandler) Submit(c *gin.Context) {
	var req submitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	res, err := h.svc.Submit(c.Request.Context(), redeemapp.SubmitInput{
		Code:       req.Code,
		Contact:    req.Contact,
		Credential: req.Credential,
		Authorized: req.Authorized,
	})
	if err != nil {
		respondRedeemError(c, err)
		return
	}
	response.Success(c, res)
}

func (h *PublicHandler) Status(c *gin.Context) {
	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	res, err := h.svc.Status(c.Request.Context(), req.Code)
	if err != nil {
		respondRedeemError(c, err)
		return
	}
	response.Success(c, res)
}

func respondRedeemError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, redeemcontract.ErrDisabled):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_disabled", nil)
	case errors.Is(err, redeemcontract.ErrMaintenance):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_maintenance", nil)
	case errors.Is(err, redeemcontract.ErrInvalidCode), errors.Is(err, redeemcontract.ErrCodeNotFound):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_invalid_code", nil)
	case errors.Is(err, redeemcontract.ErrAlreadySucceeded):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_already_succeeded", nil)
	case errors.Is(err, redeemcontract.ErrInCooldown):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_cooldown", nil)
	case errors.Is(err, redeemcontract.ErrBusy):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_busy", nil)
	case errors.Is(err, redeemcontract.ErrPendingConfirm):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_pending_confirm", nil)
	case errors.Is(err, redeemcontract.ErrInvalidContact):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_invalid_contact", nil)
	case errors.Is(err, redeemcontract.ErrInvalidCredential):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_invalid_credential", nil)
	case errors.Is(err, redeemcontract.ErrNotAuthorized):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_not_authorized", nil)
	case errors.Is(err, redeemcontract.ErrMapMissing):
		ginutil.RespondError(c, response.CodeBadRequest, "error.redeem_map_missing", nil)
	case errors.Is(err, redeemcontract.ErrNotFound):
		ginutil.RespondError(c, response.CodeNotFound, "error.redeem_not_found", nil)
	default:
		ginutil.RespondError(c, response.CodeInternal, "error.redeem_failed", err)
	}
}

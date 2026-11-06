package redeemhttp

import "github.com/gin-gonic/gin"

func RegisterPublicRoutes(public gin.IRoutes, handler *PublicHandler) {
	public.POST("/redeem/verify", handler.Verify)
	public.POST("/redeem/submit", handler.Submit)
	public.POST("/redeem/status", handler.Status)
}

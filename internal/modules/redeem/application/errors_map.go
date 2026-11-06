package application

import (
	"fmt"
	"strconv"
	"strings"
)

// UserFacingError 安全映射后的用户文案。
type UserFacingError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func MapUpstreamError(code string) UserFacingError {
	code = strings.TrimSpace(code)
	switch code {
	case "40027":
		return UserFacingError{Code: code, Message: "登录凭据已失效，请重新登录并复制最新 Session", Retryable: true}
	case "59004":
		return UserFacingError{Code: code, Message: "凭据格式或内容无效，请按教程重新复制完整内容", Retryable: true}
	case "59002":
		return UserFacingError{Code: code, Message: "该商品暂时无法开通，请联系客服", Retryable: false}
	case "59005", "59015":
		return UserFacingError{Code: code, Message: "服务暂时繁忙，请稍后再试", Retryable: true}
	case "59016":
		return UserFacingError{Code: code, Message: "排队人数较多，请稍后再试", Retryable: true}
	case "59001", "59013", "401", "403":
		return UserFacingError{Code: code, Message: "兑换通道维护中，请稍后再试", Retryable: true}
	case "59018":
		return UserFacingError{Code: code, Message: "结果确认中，请勿重复提交", Retryable: false}
	case "59007":
		return UserFacingError{Code: code, Message: "请确认已勾选授权后再提交", Retryable: true}
	default:
		if code == "" {
			return UserFacingError{Code: "", Message: "开通未成功，卡密未消耗，冷却后可重新提交或联系客服", Retryable: true}
		}
		return UserFacingError{Code: code, Message: fmt.Sprintf("开通未成功（%s），冷却后可重试或联系客服", code), Retryable: true}
	}
}

func MapUpstreamErrorInt(code int) UserFacingError {
	if code == 0 {
		return MapUpstreamError("")
	}
	return MapUpstreamError(strconv.Itoa(code))
}

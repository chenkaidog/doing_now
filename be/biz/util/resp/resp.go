package resp

import (
	"net/http"

	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"

	"github.com/cloudwego/hertz/pkg/app"
)

func respWithErr(c *app.RequestContext, data any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, &dto.CommonResp{
			Success: true,
			Code:    int(errs.Success.Code()),
			Message: errs.Success.Msg(),
			Data:    data,
		})
		return
	}

	if bizErr, ok := err.(errs.Error); ok {
		c.JSON(http.StatusOK, &dto.CommonResp{
			Success: false,
			Code:    int(bizErr.Code()),
			Message: bizErr.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, &dto.CommonResp{
		Success: false,
		Code:    int(errs.ServerError.Code()),
		Message: errs.ServerError.Msg(),
	})
}

func SuccessResp(c *app.RequestContext, data any) {
	respWithErr(c, data, nil)
}

func FailResp(c *app.RequestContext, bizErr errs.Error) {
	respWithErr(c, nil, bizErr)
}

func AbortWithErr(c *app.RequestContext, bizErr errs.Error, httpCode int) {
	c.AbortWithStatusJSON(httpCode, &dto.CommonResp{
		Success: false,
		Code:    int(bizErr.Code()),
		Message: bizErr.Msg(),
	})
}

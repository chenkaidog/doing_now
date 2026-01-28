package security

import (
	"context"
	"net/http"

	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/service/user"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/sessions"
)

func NewCredentialCheck() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		sess := sessions.Default(c)
		userID, ok1 := sess.Get("user_id").(string)
		sessCV, ok2 := sess.Get("credential_version").(uint)
		if !ok1 || userID == "" || !ok2 {
			// 用户未登录
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.CommonResp{
				Code:    int(errs.Unauthorized.Code()),
				Message: "User not logged in",
				Success: false,
			})
			return
		}
		
		currentCV, err := user.NewDefault().GetCredentialVersion(ctx, userID)
		if err != nil {
			// If user not found, they shouldn't be logged in.
			if err.Code() == errs.UserNotExist.Code() {
				c.AbortWithStatusJSON(http.StatusForbidden, dto.CommonResp{
					Code:    int(errs.SessionExpired.Code()),
					Message: "User not found",
					Success: false,
				})
				return
			}
			// For DB errors, we fail open (allow request) to avoid outage, but log it.
			hlog.CtxErrorf(ctx, "GetCredentialVersion err: %v", err)
			c.Next(ctx)
			return
		}
		
		if currentCV != sessCV {
			hlog.CtxInfof(ctx, "Credential version mismatch: session=%v, db=%v. UserID=%s", sessCV, currentCV, userID)

			c.AbortWithStatusJSON(http.StatusForbidden, dto.CommonResp{
				Code:    int(errs.SessionExpired.Code()),
				Message: "Credential has changed, please login again",
				Success: false,
			})
			return
		}

		c.Next(ctx)
	}
}

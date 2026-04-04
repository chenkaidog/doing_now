package security

import (
	"context"

	"doing_now/be/biz/model/errs"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

const (
	msgUserNotLoggedIn    = "User not logged in"
	msgUserNotFound       = "User not found"
	msgCredentialChanged  = "Credential has changed, please login again"
	logCredentialMismatch = "Credential version mismatch: session=%v, db=%v. UserID=%s"
	logGetCredentialErr   = "GetCredentialVersion err: %v"
)

func CheckCredentialValid(ctx context.Context, userID string, sessCV uint, currentCV uint, getCredentialErr errs.Error) errs.Error {
	if userID == "" {
		return errs.Unauthorized.SetMsg(msgUserNotLoggedIn)
	}
	if getCredentialErr != nil {
		if getCredentialErr.Code() == errs.UserNotExist.Code() {
			return errs.SessionExpired.SetMsg(msgUserNotFound)
		}
		hlog.CtxErrorf(ctx, logGetCredentialErr, getCredentialErr)
		return nil
	}
	if currentCV != sessCV {
		hlog.CtxInfof(ctx, logCredentialMismatch, sessCV, currentCV, userID)
		return errs.SessionExpired.SetMsg(msgCredentialChanged)
	}
	return nil
}

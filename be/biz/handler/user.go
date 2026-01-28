package handler

import (
	"context"
	"errors"
	"net/http"

	"doing_now/be/biz/middleware/jwt"
	"doing_now/be/biz/middleware/session"
	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/service/user"
	"doing_now/be/biz/util/resp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/sessions"
)

// Register 用户注册接口
//
//	@Tags			user
//	@Summary		用户注册接口
//	@Description	用户注册接口
//	@Accept			json
//	@Produce		json
//	@Param			req	body		dto.RegisterReq	true	"register request body"
//	@Success		200	{object}	dto.CommonResp{data=dto.RegisterResp}
//	@Router			/api/v1/user/register [POST]
func Register(ctx context.Context, c *app.RequestContext) {
	var req dto.RegisterReq
	if err := c.BindAndValidate(&req); err != nil {
		hlog.CtxNoticef(ctx, "BindAndValidate err: %v", err)
		resp.AbortWithErr(c, errs.ParamError.SetMsg(err.Error()), http.StatusBadRequest)
		return
	}

	u, err := user.NewDefault().Register(ctx, req.Account, req.Name, req.Password)
	if err != nil {
		resp.FailResp(c, err)
		return
	}

	resp.SuccessResp(c, dto.RegisterResp{UserID: u.UserID})
}

// Login 用户登录接口
//
//	@Tags			user
//	@Summary		用户登录接口
//	@Description	用户登录接口
//	@Accept			json
//	@Produce		json
//	@Param			req	body		dto.LoginReq	true	"login request body"
//	@Success		200	{object}	dto.CommonResp{data=dto.LoginResp}
//	@Header			200	{string}	set-cookie	"cookie"
//	@Router			/api/v1/user/login [POST]
func Login(ctx context.Context, c *app.RequestContext) {
	var req dto.LoginReq
	if err := c.BindAndValidate(&req); err != nil {
		hlog.CtxNoticef(ctx, "BindAndValidate err: %v", err)
		resp.AbortWithErr(c, errs.ParamError.SetMsg(err.Error()), http.StatusBadRequest)
		return
	}

	u, bizErr := user.NewDefault().Login(ctx, req.Account, req.Password)
	if bizErr != nil {
		resp.FailResp(c, bizErr)
		return
	}

	sess := sessions.Default(c)
	sess.Set("user_id", u.UserID)
	sess.Set("account", u.Account)
	sess.Set("name", u.Name)
	sess.Set("credential_version", u.CredentialVersion)
	if err := sess.Save(); err != nil {
		hlog.CtxErrorf(ctx, "sess.Save err: %v", err)
		resp.AbortWithErr(c, errs.ServerError.SetErr(err), http.StatusInternalServerError)
		return
	}

	payload := jwt.Payload{
		UserID:  u.UserID,
		Account: u.Account,
	}

	accessToken, expAt, jwtErr := jwt.GenerateToken(ctx, payload, sess.ID())
	if jwtErr != nil {
		resp.FailResp(c, errs.ServerError.SetErr(jwtErr))
		return
	}

	refreshToken, refreshExpAt, refreshErr := jwt.GenerateRefreshToken(ctx, sess.ID())
	if refreshErr != nil {
		resp.FailResp(c, errs.ServerError.SetErr(refreshErr))
		return
	}
	jwt.SetRefreshTokenCookie(c, refreshToken, refreshExpAt)

	resp.SuccessResp(c, dto.LoginResp{
		AccessToken: accessToken,
		ExpiresAt:   expAt,
	})
}

// RefreshToken 刷新token接口
//
//	@Tags			user
//	@Summary		刷新token接口
//	@Description	刷新token接口
//	@Accept			json
//	@Produce		json
//	@Param			req	body		dto.RefreshTokenReq	true	"refresh token request body"
//	@Success		200	{object}	dto.CommonResp{data=dto.RefreshTokenResp}
//	@Header			200	{string}	set-cookie	"cookie"
//	@Router			/api/v1/user/refresh_token [POST]
func RefreshToken(ctx context.Context, c *app.RequestContext) {
	var req dto.RefreshTokenReq
	if err := c.BindAndValidate(&req); err != nil {
		hlog.CtxNoticef(ctx, "BindAndValidate err: %v", err)
		resp.AbortWithErr(c, errs.ParamError, http.StatusBadRequest)
		return
	}

	sess := sessions.Default(c)
	sessID := sess.ID()
	if sessID == "" {
		hlog.CtxNoticef(ctx, "sessID is empty")
		resp.FailResp(c, errs.Unauthorized)
		return
	}

	refreshToken := jwt.GetRefreshTokenFromCookie(c)
	if refreshToken == "" {
		hlog.CtxNoticef(ctx, "refreshToken is empty")
		resp.FailResp(c, errs.Unauthorized)
		return
	}

	if err := jwt.RemoveRefreshToken(ctx, refreshToken, sessID); err != nil {
		if errors.Is(err, jwt.ErrRefreshTokenInvalid) {
			resp.FailResp(c, errs.Unauthorized.SetErr(err))
			return
		}
		hlog.CtxErrorf(ctx, "RemoveRefreshToken err: %v", err)
		resp.FailResp(c, errs.ServerError.SetErr(err))
		return
	}

	userID, _ := sess.Get("user_id").(string)
	account, _ := sess.Get("account").(string)
	if userID == "" || account == "" {
		hlog.CtxNoticef(ctx, "userID or account is empty")
		resp.FailResp(c, errs.Unauthorized)
		return
	}

	newAccessToken, accessExpAt, accessErr := jwt.GenerateToken(ctx,
		jwt.Payload{
			UserID:  userID,
			Account: account,
		}, sessID)
	if accessErr != nil {
		hlog.CtxErrorf(ctx, "GenerateToken err: %v", accessErr)
		resp.FailResp(c, errs.ServerError.SetErr(accessErr))
		return
	}

	newRefreshToken, refreshExpAt, refreshErr := jwt.GenerateRefreshToken(ctx, sessID)
	if refreshErr != nil {
		hlog.CtxErrorf(ctx, "GenerateRefreshToken err: %v", refreshErr)
		resp.FailResp(c, errs.ServerError.SetErr(refreshErr))
		return
	}
	jwt.SetRefreshTokenCookie(c, newRefreshToken, refreshExpAt)

	resp.SuccessResp(c, dto.RefreshTokenResp{
		AccessToken:      newAccessToken,
		ExpiresAt:        accessExpAt,
		RefreshToken:     newRefreshToken,
		RefreshExpiresAt: refreshExpAt,
	})
}

// Logout 用户登出接口
//
//	@Tags			user
//	@Summary		用户登出接口
//	@Description	用户登出接口
//	@Accept			json
//	@Produce		json
//	@Param			req				body		dto.LogoutReq	true	"logout request body"
//	@Param			Authorization	header		string			true	"jwt"
//	@Success		200				{object}	dto.CommonResp{data=dto.LogoutResp}
//	@Header			200				{string}	set-cookie	"cookie"
//	@Router			/api/v1/user/logout [POST]
func Logout(ctx context.Context, c *app.RequestContext) {
	var req dto.LogoutReq
	if err := c.BindAndValidate(&req); err != nil {
		hlog.CtxNoticef(ctx, "Logout BindAndValidate err: %v", err)
		resp.AbortWithErr(c, errs.ParamError, http.StatusBadRequest)
		return
	}

	sess := sessions.Default(c)
	sessID := sess.ID()
	if err := jwt.RemoveToken(ctx, sessID); err != nil {
		hlog.CtxErrorf(ctx, "RemoveToken err: %v", err)
	}
	if rt := jwt.GetRefreshTokenFromCookie(c); rt != "" {
		// Use session ID if available, otherwise we might not be able to delete if hash check fails.
		// Logout usually happens when user is logged in, so session exists.
		if err := jwt.RemoveRefreshToken(ctx, rt, sessID); err != nil {
			hlog.CtxErrorf(ctx, "RemoveRefreshToken err: %v", err)
		}
		jwt.ClearRefreshTokenCookie(c)
	}
	if err := session.Remove(c); err != nil {
		hlog.CtxErrorf(ctx, "RemoveSession err: %v", err)
	}
	hlog.CtxInfof(ctx, "Logout success")
	resp.SuccessResp(c, dto.LogoutResp{})
}

// GetUserInfo 获取用户信息接口
//
//	@Tags			user
//	@Summary		获取用户信息接口
//	@Description	获取用户信息接口
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string	true	"jwt"
//	@Success		200				{object}	dto.CommonResp{data=dto.GetUserInfoResp}
//	@Router			/api/v1/user/info [GET]
func GetUserInfo(ctx context.Context, c *app.RequestContext) {
	var req dto.GetUserInfoReq
	if err := c.BindAndValidate(&req); err != nil {
		hlog.CtxNoticef(ctx, "BindAndValidate err: %v", err)
		resp.AbortWithErr(c, errs.ParamError, http.StatusBadRequest)
		return
	}

	payload := jwt.GetPayload(ctx)
	if payload.UserID == "" {
		resp.FailResp(c, errs.Unauthorized)
		return
	}

	u, bizErr := user.NewDefault().GetByUserID(ctx, payload.UserID)
	if bizErr != nil {
		resp.FailResp(c, bizErr)
		return
	}

	resp.SuccessResp(c, dto.GetUserInfoResp{
		UserID:    u.UserID,
		Account:   u.Account,
		Name:      u.Name,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	})
}

// UpdateInfo 更新用户信息接口
//
//	@Tags			user
//	@Summary		更新用户信息接口
//	@Description	更新用户信息接口
//	@Accept			json
//	@Produce		json
//	@Param			req				body		dto.UpdateInfoReq	true	"update info request body"
//	@Param			Authorization	header		string				true	"jwt"
//	@Success		200				{object}	dto.CommonResp{data=dto.UpdateInfoResp}
//	@Router			/api/v1/user/update_info [POST]
func UpdateInfo(ctx context.Context, c *app.RequestContext) {
	var req dto.UpdateInfoReq
	if err := c.BindAndValidate(&req); err != nil {
		hlog.CtxNoticef(ctx, "BindAndValidate err: %v", err)
		resp.AbortWithErr(c, errs.ParamError, http.StatusBadRequest)
		return
	}

	payload := jwt.GetPayload(ctx)
	if payload.UserID == "" {
		resp.FailResp(c, errs.Unauthorized)
		return
	}

	if err := user.NewDefault().UpdateInfo(ctx, payload.UserID, req.Name); err != nil {
		resp.FailResp(c, err)
		return
	}

	resp.SuccessResp(c, dto.UpdateInfoResp{})
}

// UpdatePassword 更新密码接口
//
//	@Tags			user
//	@Summary		更新密码接口
//	@Description	更新密码接口
//	@Accept			json
//	@Produce		json
//	@Param			req				body		dto.UpdatePasswordReq	true	"update password request body"
//	@Param			Authorization	header		string					true	"jwt"
//	@Success		200				{object}	dto.CommonResp{data=dto.UpdatePasswordResp}
//	@Router			/api/v1/user/update_password [POST]
func UpdatePassword(ctx context.Context, c *app.RequestContext) {
	var req dto.UpdatePasswordReq
	if err := c.BindAndValidate(&req); err != nil {
		hlog.CtxNoticef(ctx, "BindAndValidate err: %v", err)
		resp.AbortWithErr(c, errs.ParamError, http.StatusBadRequest)
		return
	}

	payload := jwt.GetPayload(ctx)
	if payload.UserID == "" {
		resp.FailResp(c, errs.Unauthorized)
		return
	}

	if err := user.NewDefault().UpdatePassword(ctx, payload.UserID, req.OldPassword, req.NewPassword); err != nil {
		resp.FailResp(c, err)
		return
	}

	resp.SuccessResp(c, dto.UpdatePasswordResp{})
}

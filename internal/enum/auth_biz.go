package enum

const (
	OAuthClientID = "sensecraft-cli"

	AuthAPIBaseGlobal = "https://account.seeed.cc/authapi"
	AuthAPIBaseChina  = "https://account.seeed.cn/authapi"
)

const (
	BizCodeSuccess             = 0
	BizCodeServerError         = 10000
	BizCodeOAuthPending        = 17014
	BizCodeOAuthDenied         = 17015
	BizCodeOAuthExpired        = 17016
	BizCodeOAuthInvalidGrant   = 17017
	BizCodeOAuthClientNotAllow = 17018
	BizCodeParamInvalid        = 11202
)

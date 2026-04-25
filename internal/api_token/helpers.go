package apitoken

func ToTokenResponse(result *TokenResult) TokenResponse {
	var accessToken, refreshToken string
	if result.AccessToken != nil {
		accessToken = result.AccessToken.Token
	}
	if result.RefreshToken != nil {
		refreshToken = result.RefreshToken.Token
	}
	return TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

package apitoken

type TokenResult struct {
	AccessToken  *APIToken
	RefreshToken *APIToken
}

type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

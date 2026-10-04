package token

type TokenPairResponse struct {
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	RememberMe   bool   `json:"-"`
}

package models

import "time"

const DefaultLocalUserID = "legacy-local-user"

type User struct {
	ID        string    `json:"id"`
	GoogleSub string    `json:"-"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatarUrl,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AuthResponse struct {
	User *User `json:"user"`
}

type AuthMeResponse struct {
	User *User `json:"user"`
}

type LogoutResponse struct {
	Success bool `json:"success"`
}

type GoogleAuthRequest struct {
	IDToken string `json:"idToken"`
}

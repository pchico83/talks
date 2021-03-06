package model

import (
	"net/http"
	"net/mail"
)

// User is the authenticated user
type User struct {
	Model
	Email          string `gorm:"email:unique_index"`
	Token          string `gorm:"token:unique_index"`
	Mixpanel       string `gorm:"mixpanel"`
	GHInstallation *GHInstallation
	Verified       bool
}

//InviteUserRequest is the request used to invite a new user to the system
type InviteUserRequest struct {
	Email   string      `json:"email"`
	Project string      `json:"project,omitempty"`
	Role    ProjectRole `json:"role"`
}

// NewUser returns a User with an autogenerated ID and API token
func NewUser(email string) *User {
	token := GenerateRandomString(40)
	u := User{
		Email: email,
		Token: token,
	}

	return &u
}

// Validate validates that r is well formed
func (r *InviteUserRequest) Validate() *AppError {
	if r.Email == "" {
		return &AppError{Status: http.StatusBadRequest, Code: InvalidEmail}
	}

	_, err := mail.ParseAddress(r.Email)
	if err != nil {
		return &AppError{Status: http.StatusBadRequest, Code: InvalidEmail}
	}

	if r.Role == "" {
		r.Role = ProjectRoleUser
	}

	if r.Role != ProjectRoleUser && r.Role != ProjectRoleAdmin {
		return &AppError{Status: http.StatusBadRequest, Code: InvalidRole}
	}

	return nil
}

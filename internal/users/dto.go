package users

// User Request

type RegisterUserRequest struct {
	Name     string `json:"name" validate:"required,alphaspaceunicode"`
	Email    string `json:"email" valdiate:"required,email"`
	Password string `json:"password" validate:"required,password"`
}

type UpdateUserRequest struct {
	ID         string  `json:"-"`
	Name       *string `json:"name"  validate:"omitempty,alphaspaceunicode"`
	Email      *string `json:"email" validate:"omitempty,email"`
	RememberMe *bool   `json:"-"`
}

type UpdatePasswordRequest struct {
	ID          string `json:"-"`
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,password"`
}

type LoginRequest struct {
	Email      string `json:"email"       validate:"required,email"`
	Password   string `json:"password"    validate:"required"` 
	RememberMe bool   `json:"remember_me" validate:"omitempty"`
}

// User Response

type UserResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	RememberMe bool   `json:"remember_me"`
}
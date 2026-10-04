package pkg

import "testing"

type passwordFixture struct {
	Password string `validate:"password"`
}

func TestPasswordValidationEnforcesLengthAndComplexity(t *testing.T) {
	validate := NewValidator()
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{name: "valid", password: "Valid1!x", valid: true},
		{name: "too short", password: "A1!", valid: false},
		{name: "bcrypt maximum", password: "A1!" + string(make([]byte, 70)), valid: false},
		{name: "space", password: "Valid 1!", valid: false},
		{name: "missing symbol", password: "Valid123", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(passwordFixture{Password: tt.password})
			if (err == nil) != tt.valid {
				t.Fatalf("validation error = %v, valid want %v", err, tt.valid)
			}
		})
	}
}

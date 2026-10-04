package pkg

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var instance *validator.Validate

func NewValidator() *validator.Validate {
	instance = validator.New()
	registerCustomValidators(instance)
	return instance
}

func registerCustomValidators(v *validator.Validate) {
	v.RegisterValidation("alphaspaceunicode", alphaSpaceUnicode)
	v.RegisterValidation("password", validatePassword)
	v.RegisterValidation("http_url", validateHTTPURL)
}

func validateHTTPURL(fl validator.FieldLevel) bool {
	raw := strings.TrimSpace(fl.Field().String())
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func validatePassword(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	length := len([]byte(s))
	if length < 8 || length > 72 {
		return false
	}

	var hasUpper, hasNumber, hasSymbol bool

	for _, c := range s {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsDigit(c):
			hasNumber = true
		case !unicode.IsLetter(c) && !unicode.IsDigit(c) && !unicode.IsSpace(c):
			hasSymbol = true
		case unicode.IsSpace(c):
			return false
		}
	}

	return hasUpper && hasNumber && hasSymbol
}

func alphaSpaceUnicode(fl validator.FieldLevel) bool {
	for _, r := range fl.Field().String() {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

package pkg

import (
    "errors"
    "fmt"
    "strings"

    "github.com/go-playground/validator/v10"
)

type FieldMessage struct {
    Field   string
    Tag     string
    Message string
}

func ValidateStruct(v *validator.Validate, s any, customMessages ...FieldMessage) error {
    if err := v.Struct(s); err != nil {
        var validationErrors validator.ValidationErrors
        if errors.As(err, &validationErrors) {
            return fmt.Errorf("%s", formatErrors(validationErrors, customMessages))
        }
        return fmt.Errorf("invalid input")
    }
    return nil
}

func formatErrors(errs validator.ValidationErrors, customMessages []FieldMessage) string {
    var messages []string

    for _, e := range errs {
        msg := findCustomMessage(e, customMessages)
        if msg == "" {
            msg = defaultMessage(e)
        }
        messages = append(messages, msg)
    }

    return strings.Join(messages, ", ")
}

func findCustomMessage(e validator.FieldError, customMessages []FieldMessage) string {
    for _, cm := range customMessages {
        fieldMatch := cm.Field == "" || cm.Field == e.Field()
        tagMatch := cm.Tag == "" || cm.Tag == e.Tag()

        if fieldMatch && tagMatch {
            return cm.Message
        }
    }
    return ""
}

func defaultMessage(e validator.FieldError) string {
    switch e.Tag() {
        case "required":
            return fmt.Sprintf("%s is required or cannot be empty!", e.Field())
        case "email":
            return fmt.Sprintf("%s must be a valid email!", e.Field())
        case "alphaspaceunicode":
            return fmt.Sprintf("%s must be an alpha characters or unicode! and symbols are not permitted!", e.Field())
        case "uuid4":
            return fmt.Sprintf("%s must be an uuid type!", e.Field())
        case "password":
            return fmt.Sprintf("%s must be contains at least (1 uppercase, 1 number, 1 symbol) and space are not permitted!", e.Field())
        default:
            return fmt.Sprintf("%s is invalid!", e.Field())
    }
}
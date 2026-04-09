package api

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func init() {
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

func validateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var valErrors validator.ValidationErrors
	if !errors.As(err, &valErrors) {
		return map[string]string{"_": "invalid input"}
	}

	fieldErrors := make(map[string]string)
	for _, fieldError := range valErrors {
		fieldErrors[fieldName(fieldError)] = fieldMessage(fieldError)
	}
	return fieldErrors
}

func fieldName(fieldError validator.FieldError) string {
	return fieldError.Field()
}

func fieldMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "is required"
	case "gt":
		return "must be greater than " + fieldError.Param()
	case "oneof":
		return "must be one of: " + fieldError.Param()
	case "email":
		return "must be a valid email"
	case "min":
		return "must be at least " + fieldError.Param() + " characters long"
	default:
		return "is invalid"
	}
}

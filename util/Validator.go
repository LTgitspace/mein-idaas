package util

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidateStruct checks for tag-based validation errors
func ValidateStruct(payload interface{}) error {
	err := validate.Struct(payload)
	if err != nil {
		return err
	}
	return nil
}

package validator

import (
	"errors"
	"regexp"
)

var regExpID = regexp.MustCompile("^[a-zA-Z0-9]{1,32}$")

// ValidateInputObjectID validates the input object ID.
func ValidateInputObjectID(id string) error {
	if !regExpID.MatchString(id) {
		return errors.New("id is not valid")
	}
	return nil
}

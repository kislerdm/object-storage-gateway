package gateway

import (
	"errors"
	"regexp"
)

var regExpID = regexp.MustCompile("^[A-Z0-9]{1,32}$")

// ValidateObjectID validates the input object ID.
func ValidateObjectID(id string) error {
	if !regExpID.MatchString(id) {
		return errors.New("id is not valid")
	}
	return nil
}

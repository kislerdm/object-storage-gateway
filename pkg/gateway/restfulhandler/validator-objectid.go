package restfulhandler

import (
	"errors"
	"regexp"
)

var regExpID = regexp.MustCompile("^[a-zA-Z0-9]{1,32}$")

// validateInputObjectID validates the input object ID.
func validateInputObjectID(id string) error {
	if !regExpID.MatchString(id) {
		return errors.New("id is not valid")
	}
	return nil
}

package common

import (
	"github.com/pkg/errors"
)

var notFound = errors.New("resource was not found")

func IsNotFound(err error) bool {
	return errors.Is(err, notFound)
}

func IsUnknown(err error) bool {
	return err != nil && !errors.Is(err, notFound)
}

func NotFoundError(message string) error {
	return errors.Wrap(notFound, message)
}

func Wrap(err1 error, err2 error) error {
	switch {
	case err1 != nil && err2 != nil:
		return errors.Wrap(err1, err2.Error())
	case err1 != nil:
		return err1
	case err2 != nil:
		return err2
	default:
		return nil
	}
}

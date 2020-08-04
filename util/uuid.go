package util

import (
	"github.com/google/uuid"
)

func UUID() string {
	return uuid.New().String()
}

package profile

import "github.com/google/uuid"

func generateUuid() string {
	return uuid.New().String()
}

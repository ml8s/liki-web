package chat

import (
	"net/http"

	"liki-web/internal/auth"
)

func userID(r *http.Request) string {
	user, _ := auth.UserFromContext(r.Context())
	return user.ID
}

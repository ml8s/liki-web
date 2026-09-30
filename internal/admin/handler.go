package admin

import (
	"net/http"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
)

type AdminService struct{}

func NewHandler() *AdminService {
	return &AdminService{}
}

func (h *AdminService) HandleSession(user auth.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"email": user.Email})
	}
}

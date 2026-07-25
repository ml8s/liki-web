package admin

import (
	"context"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"

	"liki-web/internal/httputil"
	"liki-web/internal/naming"
)

// RegisterRoutes mounts admin handler routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, svc *naming.AdminService, onPaid func(ctx context.Context, orderID string)) {
	h := NewHandler(svc, onPaid)
	mux.HandleFunc("GET /api/admin/orders", adminOnly(h.HandleListOrders()))
	mux.HandleFunc("POST /api/admin/orders", adminOnly(h.HandleCreateOrder()))
	mux.HandleFunc("PATCH /api/admin/orders/{id}/expire", adminOnly(h.HandleUpdateExpire()))
	mux.HandleFunc("POST /api/admin/verification-codes", adminOnly(h.HandleGenerateCodes()))
	mux.HandleFunc("GET /api/admin/verification-codes", adminOnly(h.HandleListCodes()))
}

// adminOnly middleware — requires valid admin JWT.
func adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !adminAuth(r) {
			httputil.RespondError(w, http.StatusForbidden, "forbidden", "无权访问")
			return
		}
		next(w, r)
	}
}

// adminAuth verifies JWT and checks email matches ADMIN_EMAIL.
// Unlike JWTAuth, it does NOT require orderID (admin accounts have no order).
func adminAuth(r *http.Request) bool {
	cookie, err := r.Cookie("liki_token")
	if err != nil {
		return false
	}
	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
		return httputil.JWTSecret(), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}
	email, _ := claims["email"].(string)
	return email != "" && email == os.Getenv("ADMIN_EMAIL")
}

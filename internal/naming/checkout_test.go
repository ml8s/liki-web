package naming

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetiredProductHandlersFailClosed(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	tests := []struct {
		name   string
		target string
		handle func(http.ResponseWriter, *http.Request)
	}{
		{
			name:   "create order",
			target: "/api/naming/orders",
			handle: HandleCreateOrder(store),
		},
		{
			name:   "redeem code",
			target: "/api/naming/verification-codes/redeem",
			handle: HandleRedeemCode(store),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodPost, test.target, strings.NewReader(`{}`))
			response := httptest.NewRecorder()
			test.handle(response, request)
			if response.Code != http.StatusGone {
				t.Fatalf("status = %d, want %d, body: %s", response.Code, http.StatusGone, response.Body.String())
			}
		})
	}
}

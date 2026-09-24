package nofxos

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestIsSubscriptionUnavailableClassifiesEntitlementErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"payment required", &APIError{StatusCode: http.StatusPaymentRequired, Message: "402"}, true},
		{"unauthorized", &APIError{StatusCode: http.StatusUnauthorized, Message: "401"}, true},
		{"forbidden", &APIError{StatusCode: http.StatusForbidden, Message: "403"}, true},
		{"gone", &APIError{StatusCode: http.StatusGone, Message: "410"}, true},
		{"server error is transient", &APIError{StatusCode: http.StatusBadGateway, Message: "502"}, false},
		{"rate limited is transient", &APIError{StatusCode: http.StatusTooManyRequests, Message: "429"}, false},
		{"network error is transient", errors.New("connection reset"), false},
		{"wrapped entitlement error", fmt.Errorf("fetch: %w", &APIError{StatusCode: http.StatusPaymentRequired}), true},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := IsSubscriptionUnavailable(c.err); got != c.want {
			t.Fatalf("%s: IsSubscriptionUnavailable = %v, want %v", c.name, got, c.want)
		}
	}
}

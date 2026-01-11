package middleware

import "cdn-common/i18n"

// T returns a localized message for middleware responses.
func T(key string) string {
	return i18n.T(key)
}

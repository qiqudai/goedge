package controllers

import "cdn-common/i18n"

// T returns a localized message for controllers.
func T(key string) string {
	return i18n.T(key)
}

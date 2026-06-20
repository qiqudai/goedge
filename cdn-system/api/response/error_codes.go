package response

const (
	CodeSuccess            = 200
	CodeBadRequest         = 40001
	CodeUnauthorized       = 40101
	CodeForbidden          = 40301
	CodeNotFound           = 40401
	CodeConflict           = 40901
	CodeTooManyRequests    = 42901
	CodeInternalError      = 50001
	CodeBadGateway         = 50201
	CodeServiceUnavailable = 50301
	CodeConnectionLimit    = 51501
)

// NormalizeCode converts legacy code/http status into unified business code.
func NormalizeCode(raw interface{}, httpStatus int) int {
	code := parseInt(raw)
	if code == 0 {
		if httpStatus >= 400 {
			return FromHTTPStatus(httpStatus)
		}
		return CodeSuccess
	}
	if code == CodeSuccess {
		return CodeSuccess
	}
	// If already a business code (>=10000), keep.
	if code >= 10000 {
		return code
	}
	// If looks like HTTP status, map it.
	if code >= 100 && code <= 599 {
		return FromHTTPStatus(code)
	}
	// Legacy non-zero code (e.g. 1), treat as internal error.
	if httpStatus >= 400 {
		return FromHTTPStatus(httpStatus)
	}
	return CodeInternalError
}

// FromHTTPStatus maps HTTP status to business code.
func FromHTTPStatus(status int) int {
	switch status {
	case 400:
		return CodeBadRequest
	case 401:
		return CodeUnauthorized
	case 403:
		return CodeForbidden
	case 404:
		return CodeNotFound
	case 409:
		return CodeConflict
	case 429:
		return CodeTooManyRequests
	case 502:
		return CodeBadGateway
	case 503:
		return CodeServiceUnavailable
	case 515:
		return CodeConnectionLimit
	default:
		if status >= 500 {
			return CodeInternalError
		}
	}
	return CodeBadRequest
}

func parseInt(raw interface{}) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		if v == "" {
			return 0
		}
	}
	return 0
}

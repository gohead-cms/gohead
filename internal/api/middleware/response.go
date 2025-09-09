package middleware

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

type responseRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func ResponseWrapper() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// If the handler didn’t set a response, do nothing.
		respVal, hasResp := c.Get("response")
		if !hasResp {
			return
		}

		// Start from writer status (in case something set it explicitly),
		// then override with the context "status" key when present.
		statusCode := c.Writer.Status()
		if s, ok := c.Get("status"); ok {
			if v, ok := s.(int); ok {
				statusCode = v
			}
		}
		// Meta is optional
		meta, _ := c.Get("meta")

		// Format + write once, then abort the chain so nothing else changes it.
		if statusCode >= 400 {
			details, _ := c.Get("details")
			c.Status(statusCode)
			c.Abort()
			c.JSON(statusCode, gin.H{
				"error": gin.H{
					"status":  statusCode,
					"name":    getErrorName(statusCode),
					"message": respVal,
					"details": details,
				},
			})
			return
		}

		out := gin.H{"data": respVal}
		if meta != nil {
			out["meta"] = meta
		}
		c.Status(statusCode)
		c.Abort()
		c.JSON(statusCode, out)
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func getErrorName(status int) string {
	switch status {
	case http.StatusBadRequest: // 400
		return "ValidationError"
	case http.StatusUnauthorized: // 401
		return "UnauthorizedError"
	case http.StatusPaymentRequired: // 402
		return "PaymentRequiredError"
	case http.StatusForbidden: // 403
		return "ForbiddenError"
	case http.StatusNotFound: // 404
		return "NotFoundError"
	case http.StatusMethodNotAllowed: // 405
		return "MethodNotAllowedError"
	case http.StatusNotAcceptable: // 406
		return "NotAcceptableError"
	case http.StatusProxyAuthRequired: // 407
		return "ProxyAuthenticationRequiredError"
	case http.StatusRequestTimeout: // 408
		return "RequestTimeoutError"
	case http.StatusConflict: // 409
		return "ConflictError"
	case http.StatusGone: // 410
		return "GoneError"
	case http.StatusLengthRequired: // 411
		return "LengthRequiredError"
	case http.StatusPreconditionFailed: // 412
		return "PreconditionFailedError"
	case http.StatusRequestEntityTooLarge: // 413
		return "RequestEntityTooLargeError"
	case http.StatusRequestURITooLong: // 414
		return "RequestURITooLongError"
	case http.StatusUnsupportedMediaType: // 415
		return "UnsupportedMediaTypeError"
	case http.StatusRequestedRangeNotSatisfiable: // 416
		return "RequestedRangeNotSatisfiableError"
	case http.StatusExpectationFailed: // 417
		return "ExpectationFailedError"
	case http.StatusTeapot: // 418
		return "TeapotError"
	case http.StatusUnprocessableEntity: // 422
		return "UnprocessableEntityError"
	case http.StatusLocked: // 423
		return "LockedError"
	case http.StatusFailedDependency: // 424
		return "FailedDependencyError"
	case http.StatusTooEarly: // 425
		return "TooEarlyError"
	case http.StatusUpgradeRequired: // 426
		return "UpgradeRequiredError"
	case http.StatusPreconditionRequired: // 428
		return "PreconditionRequiredError"
	case http.StatusTooManyRequests: // 429
		return "TooManyRequestsError"
	case http.StatusRequestHeaderFieldsTooLarge: // 431
		return "RequestHeaderFieldsTooLargeError"
	case http.StatusUnavailableForLegalReasons: // 451
		return "UnavailableForLegalReasonsError"
	case http.StatusInternalServerError: // 500
		return "InternalServerError"
	case http.StatusNotImplemented: // 501
		return "NotImplementedError"
	case http.StatusBadGateway: // 502
		return "BadGatewayError"
	case http.StatusServiceUnavailable: // 503
		return "ServiceUnavailableError"
	case http.StatusGatewayTimeout: // 504
		return "GatewayTimeoutError"
	case http.StatusHTTPVersionNotSupported: // 505
		return "HTTPVersionNotSupportedError"
	case http.StatusVariantAlsoNegotiates: // 506
		return "VariantAlsoNegotiatesError"
	case http.StatusInsufficientStorage: // 507
		return "InsufficientStorageError"
	case http.StatusLoopDetected: // 508
		return "LoopDetectedError"
	case http.StatusNotExtended: // 510
		return "NotExtendedError"
	case http.StatusNetworkAuthenticationRequired: // 511
		return "NetworkAuthenticationRequiredError"
	default:
		// For any non-standard status code or something not listed above
		return "ServerError"
	}
}

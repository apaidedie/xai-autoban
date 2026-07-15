package classify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Classification labels (aligned with grok-inspection semantics).
const (
	Healthy          = "healthy"
	Reauth           = "reauth"
	QuotaExhausted   = "quota_exhausted"
	PermissionDenied = "permission_denied"
	ModelUnavailable = "model_unavailable"
	RateLimited      = "rate_limited"
	ProbeError       = "probe_error"
	Unknown          = "unknown"
)

// Recommended isolation / ops actions.
const (
	ActionKeep    = "keep"
	ActionBan     = "ban"
	ActionDisable = "disable"
	ActionDelete  = "delete"
	ActionEnable  = "enable"
)

type ErrorParts struct {
	Code    string
	Message string
}

type Input struct {
	Status       int
	Body         string
	Code         string
	Message      string
	RequestError string
	Disabled     bool
}

// Result is a semantic judgment of an upstream failure (or success).
type Result struct {
	Classification    string `json:"classification"`
	RecommendedAction string `json:"recommended_action"`
	Reason            string `json:"reason"`
	// Isolate=false means usage/auto_execute must not ban/disable/delete.
	Isolate bool `json:"isolate"`
	// StatusCode is the HTTP status to record (may differ from input for remaps).
	StatusCode int `json:"status_code"`
}

func lower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func containsAny(text string, needles ...string) bool {
	value := lower(text)
	for _, needle := range needles {
		if needle == "" {
			continue
		}
		if strings.Contains(value, lower(needle)) {
			return true
		}
	}
	return false
}

// IsFreeUsageExhausted reports Grok free-tier exhaustion only (not bare 429).
func IsFreeUsageExhausted(code, message string) bool {
	blob := lower(code) + " " + lower(message)
	return containsAny(blob,
		"free-usage-exhausted",
		"used all the included free usage",
		"included free usage has been exhausted",
	)
}

// ExtractError parses common xAI/OpenAI-style error JSON bodies.
func ExtractError(body string) ErrorParts {
	body = strings.TrimSpace(body)
	if body == "" {
		return ErrorParts{}
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return ErrorParts{Message: body}
	}
	code := asString(data["code"])
	message := ""
	switch errValue := data["error"].(type) {
	case map[string]any:
		if code == "" {
			code = asString(errValue["code"])
		}
		message = firstNonEmpty(asString(errValue["message"]), asString(errValue["error"]))
	case string:
		message = errValue
	}
	if message == "" {
		message = asString(data["message"])
	}
	if message == "" {
		message = body
	}
	return ErrorParts{Code: code, Message: message}
}

// Probe classifies a probe/chat/usage failure using status + body text.
func Probe(input Input) Result {
	status := input.Status
	code := strings.TrimSpace(input.Code)
	message := strings.TrimSpace(input.Message)
	if code == "" && message == "" && strings.TrimSpace(input.Body) != "" {
		parts := ExtractError(input.Body)
		code = parts.Code
		message = parts.Message
	}
	blob := lower(code) + " " + lower(message)
	disabled := input.Disabled

	if status == http.StatusUnauthorized || containsAny(blob,
		"token is expired",
		"token has been invalidated",
		"invalid_grant",
		"unauthorized",
	) {
		return Result{
			Classification:    Reauth,
			RecommendedAction: ActionDelete,
			Reason:            "auth expired or invalid",
			Isolate:           true,
			StatusCode:        statusOr(status, http.StatusUnauthorized),
		}
	}

	// Free-tier exhaustion (may appear as 429 with specific code).
	if IsFreeUsageExhausted(code, message) {
		action := ActionDisable
		if disabled {
			action = ActionKeep
		}
		return Result{
			Classification:    QuotaExhausted,
			RecommendedAction: action,
			Reason:            "free usage exhausted",
			Isolate:           true,
			StatusCode:        statusOr(status, http.StatusPaymentRequired),
		}
	}

	// Bare temporary rate limit: isolate for scheduling, but recommend keep (not disable).
	if status == http.StatusTooManyRequests {
		return Result{
			Classification:    RateLimited,
			RecommendedAction: ActionBan,
			Reason:            "temporary rate limit (HTTP 429)",
			Isolate:           true,
			StatusCode:        http.StatusTooManyRequests,
		}
	}

	if status == http.StatusPaymentRequired || status == http.StatusForbidden || containsAny(blob,
		"permission-denied",
		"chat endpoint is denied",
		"deactivated",
		"suspended",
		"banned",
	) {
		action := ActionDisable
		if disabled {
			action = ActionKeep
		}
		reason := "permission denied"
		if status > 0 {
			reason = fmt.Sprintf("%s (HTTP %d)", reason, status)
		}
		sc := status
		if sc == 0 {
			sc = http.StatusForbidden
		}
		return Result{
			Classification:    PermissionDenied,
			RecommendedAction: action,
			Reason:            reason,
			Isolate:           true,
			StatusCode:        sc,
		}
	}

	if status == http.StatusNotFound || containsAny(blob, "not-found", "does not exist", "no access to it") {
		return Result{
			Classification:    ModelUnavailable,
			RecommendedAction: ActionKeep,
			Reason:            "model unavailable",
			Isolate:           false,
			StatusCode:        statusOr(status, http.StatusNotFound),
		}
	}

	if status >= 200 && status < 300 {
		action := ActionKeep
		if disabled {
			action = ActionEnable
		}
		return Result{
			Classification:    Healthy,
			RecommendedAction: action,
			Reason:            "ok",
			Isolate:           false,
			StatusCode:        status,
		}
	}

	if strings.TrimSpace(input.RequestError) != "" || status > 0 {
		reason := strings.TrimSpace(input.RequestError)
		if reason == "" {
			reason = "probe failed"
			if status > 0 {
				reason = fmt.Sprintf("%s (HTTP %d)", reason, status)
			}
		}
		// Unknown HTTP failures: isolate only for classic auth/payment statuses.
		isolate := status == 401 || status == 402 || status == 403 || status == 429
		return Result{
			Classification:    ProbeError,
			RecommendedAction: ActionKeep,
			Reason:            reason,
			Isolate:           isolate,
			StatusCode:        status,
		}
	}

	return Result{
		Classification:    Unknown,
		RecommendedAction: ActionKeep,
		Reason:            "unclassified",
		Isolate:           false,
		StatusCode:        status,
	}
}

func statusOr(status, fallback int) int {
	if status > 0 {
		return status
	}
	return fallback
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

package pluginapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

const (
	SchedulerBuiltinRoundRobin = "round-robin"
	SchedulerBuiltinFillFirst  = "fill-first"
)

type Metadata struct {
	Name             string
	Version          string
	Author           string
	GitHubRepository string
	Logo             string
	ConfigFields     []ConfigField
}

type ConfigFieldType string

const (
	ConfigFieldTypeInteger ConfigFieldType = "integer"
	ConfigFieldTypeString  ConfigFieldType = "string"
	ConfigFieldTypeEnum    ConfigFieldType = "enum"
	ConfigFieldTypeBoolean ConfigFieldType = "boolean"
)

type ConfigField struct {
	Name        string
	Type        ConfigFieldType
	EnumValues  []string
	Description string
}

type SchedulerPickRequest struct {
	Plugin     Metadata
	Provider   string
	Providers  []string
	Model      string
	Stream     bool
	Options    SchedulerOptions
	Candidates []SchedulerAuthCandidate
}

type SchedulerOptions struct {
	Headers  map[string][]string
	Metadata map[string]any
}

type SchedulerAuthCandidate struct {
	ID         string
	Provider   string
	Priority   int
	Status     string
	Attributes map[string]string
	Metadata   map[string]any
}

type SchedulerPickResponse struct {
	AuthID          string
	DelegateBuiltin string
	Handled         bool
}

type ManagementRegistrationResponse struct {
	Routes    []ManagementRoute
	Resources []ResourceRoute
}

type ManagementRoute struct {
	Method      string
	Path        string
	Menu        string
	Description string
}

type ResourceRoute struct {
	Path        string
	Menu        string
	Description string
}

type ManagementRequest struct {
	Method         string
	Path           string
	Headers        http.Header
	Query          url.Values
	Body           []byte
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type ManagementResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

type UsageRecord struct {
	Provider        string
	ExecutorType    string
	Model           string
	Alias           string
	APIKey          string
	AuthID          string
	AuthIndex       string
	AuthType        string
	Source          string
	ReasoningEffort string
	ServiceTier     string
	RequestedAt     time.Time
	Latency         time.Duration
	TTFT            time.Duration
	Failed          bool
	Failure         UsageFailure
	Detail          UsageDetail
	ResponseHeaders http.Header
}

type UsageFailure struct {
	StatusCode int
	Body       string
}

type UsageDetail struct {
	InputTokens         int64
	OutputTokens        int64
	ReasoningTokens     int64
	CachedTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	TotalTokens         int64
}

type HostAuthFileEntry struct {
	ID             string    `json:"id,omitempty"`
	AuthIndex      string    `json:"auth_index,omitempty"`
	Name           string    `json:"name"`
	Type           string    `json:"type,omitempty"`
	Provider       string    `json:"provider,omitempty"`
	Label          string    `json:"label,omitempty"`
	Status         string    `json:"status,omitempty"`
	StatusMessage  string    `json:"status_message,omitempty"`
	Disabled       bool      `json:"disabled,omitempty"`
	Unavailable    bool      `json:"unavailable,omitempty"`
	RuntimeOnly    bool      `json:"runtime_only,omitempty"`
	Source         string    `json:"source,omitempty"`
	Path           string    `json:"path,omitempty"`
	Email          string    `json:"email,omitempty"`
	Priority       int       `json:"priority,omitempty"`
	Note           string    `json:"note,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	NextRetryAfter time.Time `json:"next_retry_after,omitempty"`
}

type HostAuthListResponse struct {
	Files []HostAuthFileEntry `json:"files"`
}

type HostAuthGetRequest struct {
	AuthIndex string `json:"auth_index"`
}

type HostAuthGetResponse struct {
	AuthIndex string          `json:"auth_index"`
	Name      string          `json:"name,omitempty"`
	Path      string          `json:"path,omitempty"`
	JSON      json.RawMessage `json:"json"`
}

type HostAuthSaveRequest struct {
	Name string          `json:"name"`
	JSON json.RawMessage `json:"json"`
}

type HostAuthSaveResponse struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type HTTPRequest struct {
	Method  string      `json:"Method"`
	URL     string      `json:"URL"`
	Headers http.Header `json:"Headers,omitempty"`
	Body    []byte      `json:"Body,omitempty"`
}

type HTTPResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers,omitempty"`
	Body       []byte      `json:"Body,omitempty"`
}

type HostLogRequest struct {
	Level   string         `json:"level,omitempty"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

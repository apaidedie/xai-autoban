package host

import (
	"encoding/json"
	"fmt"

	"xai-autoban/cpasdk/pluginabi"
	"xai-autoban/cpasdk/pluginapi"
)

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CallFn is set by CGO wiring in main.go when the plugin initializes.
var CallFn func(method string, request []byte) ([]byte, error)

type Client interface {
	AuthList() ([]pluginapi.HostAuthFileEntry, error)
	AuthGet(authIndex string) (pluginapi.HostAuthGetResponse, error)
	AuthSave(name string, raw json.RawMessage) (pluginapi.HostAuthSaveResponse, error)
	HTTPDo(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error)
	Log(level, message string, fields map[string]any) error
}

type Real struct{}

func (h Real) call(method string, request any) (json.RawMessage, error) {
	if CallFn == nil {
		return nil, fmt.Errorf("host callbacks unavailable")
	}
	var reqBytes []byte
	var err error
	if request == nil {
		reqBytes = []byte("{}")
	} else {
		reqBytes, err = json.Marshal(request)
		if err != nil {
			return nil, err
		}
	}
	raw, err := CallFn(method, reqBytes)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host call %s returned not ok", method)
	}
	return env.Result, nil
}

func (h Real) AuthList() ([]pluginapi.HostAuthFileEntry, error) {
	raw, err := h.call(pluginabi.MethodHostAuthList, map[string]any{})
	if err != nil {
		return nil, err
	}
	var resp pluginapi.HostAuthListResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return resp.Files, nil
}

func (h Real) AuthGet(authIndex string) (pluginapi.HostAuthGetResponse, error) {
	raw, err := h.call(pluginabi.MethodHostAuthGet, pluginapi.HostAuthGetRequest{AuthIndex: authIndex})
	if err != nil {
		return pluginapi.HostAuthGetResponse{}, err
	}
	var resp pluginapi.HostAuthGetResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return pluginapi.HostAuthGetResponse{}, err
	}
	return resp, nil
}

func (h Real) AuthSave(name string, body json.RawMessage) (pluginapi.HostAuthSaveResponse, error) {
	raw, err := h.call(pluginabi.MethodHostAuthSave, pluginapi.HostAuthSaveRequest{Name: name, JSON: body})
	if err != nil {
		return pluginapi.HostAuthSaveResponse{}, err
	}
	var resp pluginapi.HostAuthSaveResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return pluginapi.HostAuthSaveResponse{}, err
	}
	return resp, nil
}

func (h Real) HTTPDo(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
	raw, err := h.call(pluginabi.MethodHostHTTPDo, req)
	if err != nil {
		return pluginapi.HTTPResponse{}, err
	}
	var resp pluginapi.HTTPResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return pluginapi.HTTPResponse{}, err
	}
	return resp, nil
}

func (h Real) Log(level, message string, fields map[string]any) error {
	_, err := h.call(pluginabi.MethodHostLog, pluginapi.HostLogRequest{Level: level, Message: message, Fields: fields})
	return err
}

type Stub struct {
	Files   []pluginapi.HostAuthFileEntry
	JSONBy  map[string]json.RawMessage
	Saves   []pluginapi.HostAuthSaveRequest
	HTTPFn  func(pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error)
	ListErr error
}

func (s *Stub) AuthList() ([]pluginapi.HostAuthFileEntry, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	return s.Files, nil
}

func (s *Stub) AuthGet(authIndex string) (pluginapi.HostAuthGetResponse, error) {
	raw, ok := s.JSONBy[authIndex]
	if !ok {
		return pluginapi.HostAuthGetResponse{}, fmt.Errorf("missing auth %s", authIndex)
	}
	return pluginapi.HostAuthGetResponse{AuthIndex: authIndex, Name: authIndex + ".json", JSON: raw}, nil
}

func (s *Stub) AuthSave(name string, body json.RawMessage) (pluginapi.HostAuthSaveResponse, error) {
	s.Saves = append(s.Saves, pluginapi.HostAuthSaveRequest{Name: name, JSON: body})
	return pluginapi.HostAuthSaveResponse{Name: name}, nil
}

func (s *Stub) HTTPDo(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
	if s.HTTPFn != nil {
		return s.HTTPFn(req)
	}
	return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[]}`)}, nil
}

func (s *Stub) Log(level, message string, fields map[string]any) error { return nil }

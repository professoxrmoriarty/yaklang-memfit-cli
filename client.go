package aiclientv2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AgentClient struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
	totpCode   string
}

func NewAgentClient(baseURL string) *AgentClient {
	return &AgentClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 0,
		},
	}
}

func (c *AgentClient) SetAuthToken(token string) {
	c.authToken = token
}

func (c *AgentClient) SetTOTPCode(code string) {
	c.totpCode = code
}

func (c *AgentClient) addAuth(req *http.Request) {
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	if c.totpCode != "" {
		req.Header.Set("X-TOTP-Code", c.totpCode)
	}
}

func (c *AgentClient) GetSetting() (*SettingResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/setting", nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var setting SettingResponse
	if err := json.NewDecoder(resp.Body).Decode(&setting); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &setting, nil
}

func (c *AgentClient) UpdateSetting(s SettingRequest) (*SettingResponse, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/setting", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var setting SettingResponse
	if err := json.NewDecoder(resp.Body).Decode(&setting); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &setting, nil
}

func (c *AgentClient) ListAIProviders() ([]ProviderOption, error) {
	req, err := http.NewRequest("POST", c.baseURL+"/setting/providers/get", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	providers := toAnySlice(raw["Providers"])
	if len(providers) == 0 {
		providers = toAnySlice(raw["providers"])
	}

	result := make([]ProviderOption, 0, len(providers))
	for _, item := range providers {
		row := toMap(item)
		if row == nil {
			continue
		}
		opt := ProviderOption{
			ID: toInt64(firstNonNil(row["Id"], row["id"])),
		}

		cfg := toMap(firstNonNil(row["Config"], row["config"]))
		if cfg != nil {
			opt.Type = strings.TrimSpace(toString(firstNonNil(cfg["Type"], cfg["type"])))
			opt.Domain = strings.TrimSpace(toString(firstNonNil(cfg["Domain"], cfg["domain"])))
			opt.Disabled = toBool(firstNonNil(cfg["Disabled"], cfg["disabled"]))
			opt.Identifier = strings.TrimSpace(toString(firstNonNil(
				cfg["UserIdentifier"],
				cfg["user_identifier"],
				cfg["Identifier"],
				cfg["identifier"],
			)))
		} else {
			opt.Type = strings.TrimSpace(toString(firstNonNil(row["Type"], row["type"])))
			opt.Domain = strings.TrimSpace(toString(firstNonNil(row["Domain"], row["domain"])))
			opt.Disabled = toBool(firstNonNil(row["Disabled"], row["disabled"]))
			opt.Identifier = strings.TrimSpace(toString(firstNonNil(row["Identifier"], row["identifier"])))
		}

		if opt.ID == 0 && opt.Type == "" {
			continue
		}
		result = append(result, opt)
	}
	return result, nil
}

func (c *AgentClient) ListAIModels(config string) ([]string, error) {
	config = strings.TrimSpace(config)
	if config == "" {
		return nil, fmt.Errorf("config is required")
	}

	body, err := json.Marshal(map[string]string{"Config": config})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/setting/aimodels/get", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	modelRows := toAnySlice(raw["ModelName"])
	if len(modelRows) == 0 {
		modelRows = toAnySlice(raw["model_name"])
	}
	if len(modelRows) == 0 {
		modelRows = toAnySlice(raw["models"])
	}

	seen := make(map[string]struct{})
	models := make([]string, 0, len(modelRows))
	for _, row := range modelRows {
		name := strings.TrimSpace(toString(row))
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		models = append(models, name)
	}
	return models, nil
}

func (c *AgentClient) ListAIFocusModes() ([]FocusModeOption, error) {
	req, err := http.NewRequest("POST", c.baseURL+"/setting/aifocus/get", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	rows := toAnySlice(raw["Data"])
	if len(rows) == 0 {
		rows = toAnySlice(raw["data"])
	}
	if len(rows) == 0 {
		rows = toAnySlice(raw["focus_modes"])
	}

	result := make([]FocusModeOption, 0, len(rows))
	for _, row := range rows {
		if text := strings.TrimSpace(toString(row)); text != "" {
			result = append(result, FocusModeOption{Name: text})
			continue
		}
		item := toMap(row)
		if item == nil {
			continue
		}
		name := strings.TrimSpace(toString(firstNonNil(item["Name"], item["name"])))
		verboseName := strings.TrimSpace(toString(firstNonNil(item["VerboseName"], item["verbose_name"])))
		verboseNameZh := strings.TrimSpace(toString(firstNonNil(item["VerboseNameZh"], item["verbose_name_zh"])))
		desc := strings.TrimSpace(toString(firstNonNil(item["Description"], item["description"])))

		if name == "" {
			name = verboseName
		}
		if name == "" {
			name = verboseNameZh
		}
		if name == "" {
			continue
		}

		result = append(result, FocusModeOption{
			Name:          name,
			VerboseName:   verboseName,
			VerboseNameZh: verboseNameZh,
			Description:   desc,
		})
	}
	return result, nil
}

func (c *AgentClient) CreateSession(runID, taskID string, params AIParams) (*CreateSessionResponse, error) {
	body, err := json.Marshal(CreateSessionRequest{
		RunID:  runID,
		TaskID: taskID,
		Params: params,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/session", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func (c *AgentClient) GetResult(runID string) (*RunResult, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/run/"+runID, nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var result RunResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

type SSEEvent struct {
	Event string
	Data  string
}

func (c *AgentClient) StreamEvents(ctx context.Context, runID string, since int64, eventCh chan<- SSEEvent) error {
	url := fmt.Sprintf("%s/run/%s/events", c.baseURL, runID)
	if since > 0 {
		url += fmt.Sprintf("?since=%d", since)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return parseErrorResponse(resp)
	}

	go func() {
		defer resp.Body.Close()
		defer close(eventCh)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

		var currentEvent, currentData string
		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				if currentData != "" {
					evt := currentEvent
					if evt == "" {
						evt = extractTypeFromJSON(currentData)
					}
					eventCh <- SSEEvent{Event: evt, Data: currentData}
				}
				currentEvent = ""
				currentData = ""
				continue
			}

			if strings.HasPrefix(line, "event: ") {
				currentEvent = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				currentData = strings.TrimPrefix(line, "data: ")
			}
		}
	}()

	return nil
}

func (c *AgentClient) Run(runID string, runReq PushEventRequest) error {
	body, err := json.Marshal(runReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/run/%s", c.baseURL, runID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return parseErrorResponse(resp)
	}
	return nil
}

func (c *AgentClient) PushEvent(runID string, pushReq PushEventRequest) error {
	body, err := json.Marshal(pushReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/run/%s/events/push", c.baseURL, runID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseErrorResponse(resp)
	}
	return nil
}

func (c *AgentClient) RunAndStream(ctx context.Context, runID string, runReq PushEventRequest, eventCh chan<- SSEEvent) error {
	events := make(chan SSEEvent, 256)
	if err := c.StreamEvents(ctx, runID, 0, events); err != nil {
		return err
	}

	go func() {
		defer close(eventCh)

		triggered := false
		for evt := range events {
			typ := evt.Event
			if typ == "" {
				typ = extractTypeFromJSON(evt.Data)
			}
			if typ == "listener_ready" && !triggered {
				triggered = true
				if err := c.Run(runID, runReq); err != nil {
					eventCh <- SSEEvent{
						Event: "error",
						Data:  toClientErrorEvent(err),
					}
					return
				}
			}
			eventCh <- evt
		}

		if !triggered {
			eventCh <- SSEEvent{
				Event: "error",
				Data:  toClientErrorEvent(fmt.Errorf("stream closed before listener_ready")),
			}
		}
	}()

	return nil
}

func (c *AgentClient) CancelRun(runID string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/run/%s/cancel", c.baseURL, runID), nil)
	if err != nil {
		return err
	}
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseErrorResponse(resp)
	}
	return nil
}

func parseErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp ErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
		return fmt.Errorf("[%d] %s", resp.StatusCode, errResp.Message)
	}
	return fmt.Errorf("[%d] %s", resp.StatusCode, string(body))
}

func GenerateTaskID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

func extractTypeFromJSON(data string) string {
	var m struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(data), &m) == nil && m.Type != "" {
		return m.Type
	}
	return "message"
}

func toClientErrorEvent(err error) string {
	payload, _ := json.Marshal(map[string]string{
		"type":    "error",
		"content": err.Error(),
	})
	return string(payload)
}

func toMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func toAnySlice(v any) []any {
	items, _ := v.([]any)
	return items
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	case json.Number:
		return val.String()
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case uint64:
		return int64(val)
	case uint32:
		return int64(val)
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	case json.Number:
		n, _ := val.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		return n
	default:
		return 0
	}
}

func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(val))
		return parsed
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	default:
		return false
	}
}

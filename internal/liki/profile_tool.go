package liki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
)

type sessionBindingKey struct{}

// sessionBinding carries the authenticated session identity into tool
// execution so a server-side tool can prove which profile it is allowed to
// touch. Set by Service.SendMessage before the runner runs.
type sessionBinding struct {
	UserID    string
	SessionID string
	ProfileID string
}

func withSessionBinding(ctx context.Context, binding sessionBinding) context.Context {
	return context.WithValue(ctx, sessionBindingKey{}, binding)
}

func sessionBindingFrom(ctx context.Context) (sessionBinding, bool) {
	binding, ok := ctx.Value(sessionBindingKey{}).(sessionBinding)
	return binding, ok
}

var birthTimeRe = regexp.MustCompile(`^\d{2}:\d{2}$`)

// UpdateProfileBirthTimeHandler is a server-side tool that persists a
// calibrated birth time onto the session's bound profile by calling the same
// HTTP API the frontend uses — GET then PATCH /api/profiles/{id} — with the
// caller's session cookie. It is deliberately identical in spirit to the
// agent's engine RPC tools: every write goes over the network to the
// authoritative service.
func UpdateProfileBirthTimeHandler(selfURL, cookieName string, client *http.Client) func(context.Context, json.RawMessage) (json.RawMessage, error) {
	if client == nil {
		client = http.DefaultClient
	}
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		// I4：写工具统一前置 guard——校验会话绑定 + 凭证，返回可用的档案身份。
		binding, token, err := requireBoundProfile(ctx)
		if err != nil {
			return nil, err
		}
		var args struct {
			ProfileID      string `json:"profile_id"`
			BirthTime      string `json:"birth_time"`
			BirthTimeKnown string `json:"birth_time_known"`
		}
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, fmt.Errorf("update_profile_birth_time: %w", err)
		}
		if args.ProfileID != binding.ProfileID {
			return nil, fmt.Errorf("只能更新当前会话绑定的档案")
		}
		if args.BirthTimeKnown != "known" && args.BirthTimeKnown != "approximate" {
			return nil, fmt.Errorf("birth_time_known 必须为 known 或 approximate")
		}
		if !birthTimeRe.MatchString(args.BirthTime) {
			return nil, fmt.Errorf("birth_time 必须为 HH:mm")
		}

		// 1) GET current profile for its version and full field set.
		current, err := callProfileAPI(ctx, client, http.MethodGet, selfURL, cookieName, token, binding.ProfileID, nil)
		if err != nil {
			return nil, err
		}
		// 2) PATCH the full input with the calibrated birth time — the exact
		//    request the frontend edit form sends.
		body := map[string]any{
			"expected_version":    current.Version,
			"nickname":            current.Nickname,
			"name":                current.Name,
			"gender":              current.Gender,
			"calendar":            current.Calendar,
			"birth_date":          current.BirthDate,
			"birth_time":          args.BirthTime,
			"birth_time_known":    args.BirthTimeKnown,
			"birth_place":         current.BirthPlace,
			"note":                current.Note,
			"third_party_consent": current.ThirdPartyConsent,
		}
		updated, err := callProfileAPI(ctx, client, http.MethodPatch, selfURL, cookieName, token, binding.ProfileID, body)
		if err != nil {
			return nil, err
		}
		out, err := json.Marshal(map[string]any{
			"status": "updated", "birth_time": updated.BirthTime, "birth_time_known": updated.BirthTimeKnown,
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
}

// requireBoundProfile is the unified guard every server-side write tool must
// pass through: it proves the session is bound to a profile and carries the
// caller's session token, then returns the identity for the update.
func requireBoundProfile(ctx context.Context) (sessionBinding, string, error) {
	binding, ok := sessionBindingFrom(ctx)
	if !ok || binding.UserID == "" || binding.ProfileID == "" {
		return sessionBinding{}, "", fmt.Errorf("当前会话未绑定档案，无法更新")
	}
	token, ok := auth.SessionTokenFromContext(ctx)
	if !ok || token == "" {
		return sessionBinding{}, "", fmt.Errorf("缺少会话凭证，无法更新档案")
	}
	return binding, token, nil
}

type profilePayload struct {
	Version           int    `json:"version"`
	Nickname          string `json:"nickname"`
	Name              string `json:"name"`
	Gender            string `json:"gender"`
	Calendar          string `json:"calendar"`
	BirthDate         string `json:"birth_date"`
	BirthTime         string `json:"birth_time"`
	BirthTimeKnown    string `json:"birth_time_known"`
	BirthPlace        string `json:"birth_place"`
	Note              string `json:"note"`
	ThirdPartyConsent bool   `json:"third_party_consent"`
}

func callProfileAPI(ctx context.Context, client *http.Client, method, selfURL, cookieName, token, profileID string, body any) (profilePayload, error) {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return profilePayload{}, err
		}
		payload = bytes.NewReader(raw)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	base := strings.TrimRight(selfURL, "/")
	u, _ := url.ParseRequestURI(base + "/api/profiles/" + profileID)
	url := u.String()
	req, err := http.NewRequestWithContext(reqCtx, method, url, payload)
	if err != nil {
		return profilePayload{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: cookieName, Value: token})

	resp, err := client.Do(req)
	if err != nil {
		return profilePayload{}, fmt.Errorf("更新档案失败：%w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return profilePayload{}, err
	}
	if resp.StatusCode == http.StatusConflict {
		return profilePayload{}, fmt.Errorf("档案刚被其他设备修改，请告知用户刷新后重试")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return profilePayload{}, fmt.Errorf("操作太频繁，请稍后重试")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var env struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &env); err == nil && env.Error.Code != "" {
			return profilePayload{}, fmt.Errorf("更新档案失败（%s）", env.Error.Code)
		}
		return profilePayload{}, fmt.Errorf("更新档案失败（HTTP %d）", resp.StatusCode)
	}
	var env struct {
		Data profilePayload `json:"data"`
	}
	if err := json.Unmarshal(respBody, &env); err != nil {
		return profilePayload{}, fmt.Errorf("更新档案响应异常：%w", err)
	}
	return env.Data, nil
}

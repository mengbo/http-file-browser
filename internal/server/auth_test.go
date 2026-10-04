package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mengbo/http-file-browser/web"
)

const testToken = "test-token-abcdefghijklmnopqrstuvwxyz012345"

// authHandler 构造启用认证的根 handler（登录页取自内嵌的 web.FS）。
func authHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, err := NewHandler(t.TempDir(), web.FS, Auth{Token: testToken})
	if err != nil {
		t.Fatalf("构造启用认证的 handler 失败：%v", err)
	}
	return handler
}

// serve 以给定 Cookie 请求 handler，返回响应记录器。
func serve(handler http.Handler, method, target string, cookie *http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func validCookie() *http.Cookie {
	return &http.Cookie{Name: accessCookieName, Value: testToken}
}

// TestGenerateTokenIsNonEmptyAndUnique 覆盖 Access credential 的「每次启动唯一」。
func TestGenerateTokenIsNonEmptyAndUnique(t *testing.T) {
	first, err := GenerateToken()
	if err != nil {
		t.Fatalf("生成凭证失败：%v", err)
	}
	if first == "" {
		t.Fatal("生成的凭证为空")
	}
	second, err := GenerateToken()
	if err != nil {
		t.Fatalf("再次生成凭证失败：%v", err)
	}
	if first == second {
		t.Errorf("两次生成的凭证相同：%q", first)
	}
}

// TestTokenMatches 覆盖正确凭证通过、错误与前缀凭证被拒（task 2.1）。
func TestTokenMatches(t *testing.T) {
	if !tokenMatches(testToken, testToken) {
		t.Error("完全相同的凭证应判定通过")
	}
	if tokenMatches(testToken, "wrong") {
		t.Error("错误凭证应判定拒绝")
	}
	if tokenMatches(testToken, testToken[:len(testToken)-1]) {
		t.Error("前缀凭证应判定拒绝")
	}
	if tokenMatches("", "") {
		t.Error("未启用认证（期望凭证为空）时任何输入都应判定拒绝")
	}
}

// TestUnauthorizedCodeMapsToStatus 断言 unauthorized 映射为 401（task 2.2）。
func TestUnauthorizedCodeMapsToStatus(t *testing.T) {
	if got := codeStatus[codeUnauthorized]; got != http.StatusUnauthorized {
		t.Errorf("codeStatus[unauthorized] = %d，期望 %d", got, http.StatusUnauthorized)
	}
}

// TestAPIWithoutAuthenticationReturnsUnauthorizedEnvelope 覆盖 Scenario：
// API request without authentication（401 + JSON 信封 + unauthorized）。
func TestAPIWithoutAuthenticationReturnsUnauthorizedEnvelope(t *testing.T) {
	recorder := serve(authHandler(t), http.MethodGet, "/api/list", nil)

	detail, status := decodeError(t, recorder)
	if status != http.StatusUnauthorized {
		t.Errorf("状态码 = %d，期望 %d", status, http.StatusUnauthorized)
	}
	if detail.Code != codeUnauthorized {
		t.Errorf("error.code = %q，期望 %q", detail.Code, codeUnauthorized)
	}
}

// TestAuthenticationFailureIsDistinguishable 覆盖 Scenario：
// Authentication failure is distinguishable——认证失败与内容失败的错误标识不同。
func TestAuthenticationFailureIsDistinguishable(t *testing.T) {
	handler := authHandler(t)

	_, authStatus := decodeError(t, serve(handler, http.MethodGet, "/api/list", nil))
	detail, contentStatus := decodeError(t, serve(handler, http.MethodGet, "/api/nope", validCookie()))

	if authStatus != http.StatusUnauthorized {
		t.Errorf("未认证状态码 = %d，期望 %d", authStatus, http.StatusUnauthorized)
	}
	if detail.Code == codeUnauthorized {
		t.Errorf("内容失败的标识不应等于认证失败标识 %q", codeUnauthorized)
	}
	if contentStatus == http.StatusUnauthorized {
		t.Errorf("内容失败不应返回 401")
	}
}

// TestAPIWithValidCredentialSucceeds 覆盖 Scenario：Remote request with a valid credential。
func TestAPIWithValidCredentialSucceeds(t *testing.T) {
	recorder := serve(authHandler(t), http.MethodGet, "/api/health", validCookie())
	if recorder.Code != http.StatusOK {
		t.Errorf("携带有效凭证的状态码 = %d，期望 %d（body = %q）", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

// TestLoopbackAuthDisabledAllowsRequest 覆盖 Scenario：
// Loopback-only binding does not require authentication。
func TestLoopbackAuthDisabledAllowsRequest(t *testing.T) {
	handler, err := NewHandler(t.TempDir(), web.FS, Auth{})
	if err != nil {
		t.Fatalf("构造未启用认证的 handler 失败：%v", err)
	}
	recorder := serve(handler, http.MethodGet, "/api/health", nil)
	if recorder.Code != http.StatusOK {
		t.Errorf("回环配置下无凭证的状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
	}
}

// TestValidTokenExchangeSetsCookieAndRedirects 覆盖 Scenario：
// Acquiring a session with the credential 与 The credential does not remain in the browsed location。
func TestValidTokenExchangeSetsCookieAndRedirects(t *testing.T) {
	recorder := serve(authHandler(t), http.MethodGet, "/?path=sub&token="+testToken, nil)

	if recorder.Code != http.StatusFound {
		t.Fatalf("状态码 = %d，期望 %d（body = %q）", recorder.Code, http.StatusFound, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "/?path=sub" {
		t.Errorf("Location = %q，期望去掉 token、保留 path 的 /?path=sub", location)
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie 数量 = %d，期望 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != accessCookieName || cookie.Value != testToken {
		t.Errorf("Cookie = %s=%s，期望 %s=%s", cookie.Name, cookie.Value, accessCookieName, testToken)
	}
	if cookie.Path != "/" {
		t.Errorf("Cookie Path = %q，期望 /", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Error("Cookie 应设置 HttpOnly")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("Cookie SameSite = %v，期望 Strict", cookie.SameSite)
	}
	if cookie.Secure {
		t.Error("纯 HTTP 下 Cookie 不应设置 Secure")
	}
	if cookie.MaxAge != 0 {
		t.Errorf("Cookie MaxAge = %d，期望 0（会话 Cookie）", cookie.MaxAge)
	}
}

// TestInvalidTokenIsRejectedWithoutCookie 覆盖 Scenario：Remote request with an invalid credential。
func TestInvalidTokenIsRejectedWithoutCookie(t *testing.T) {
	recorder := serve(authHandler(t), http.MethodGet, "/?token=wrong", nil)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("无效 token 状态码 = %d，期望 %d", recorder.Code, http.StatusUnauthorized)
	}
	if raw := recorder.Header().Get("Set-Cookie"); raw != "" {
		t.Errorf("无效 token 不应种 Cookie，实际 Set-Cookie = %q", raw)
	}
}

// TestUnauthenticatedPageReturnsLoginEntry 覆盖 Scenario：Page request without authentication。
func TestUnauthenticatedPageReturnsLoginEntry(t *testing.T) {
	recorder := serve(authHandler(t), http.MethodGet, "/", nil)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("未认证页面状态码 = %d，期望 %d", recorder.Code, http.StatusUnauthorized)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q，期望 HTML 登录页", contentType)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `name="token"`) {
		t.Errorf("登录页缺少 token 输入字段：%q", body)
	}
	if !strings.Contains(body, "<form") {
		t.Errorf("登录页缺少表单：%q", body)
	}
	// 自包含：不得引用任何外部资源，否则那些请求同样未认证、拿不到资源（design D7）。
	if strings.Contains(body, "<link") || strings.Contains(body, "<script") {
		t.Errorf("登录页引用了外部资源：%q", body)
	}
	// 提供的是登录入口，而非所请求页面的内容。
	if strings.Contains(body, "/app.js") {
		t.Errorf("未认证页面不应提供应用页面内容：%q", body)
	}
}

// TestPageWithValidCookieSucceeds 覆盖 Scenario：Subsequent requests need no credential。
func TestPageWithValidCookieSucceeds(t *testing.T) {
	recorder := serve(authHandler(t), http.MethodGet, "/", validCookie())

	if recorder.Code != http.StatusOK {
		t.Fatalf("携带有效 Cookie 的页面状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "/app.js") {
		t.Errorf("携带有效 Cookie 时应得到应用页面：%q", recorder.Body.String())
	}
}

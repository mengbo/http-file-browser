package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

// Auth 是根 handler 的认证配置。零值（Token 为空）表示不启用认证——对应仅监听
// 回环地址的启动方式；非空表示启用，Token 是本次运行唯一、进程内有效的访问凭证。
type Auth struct {
	Token string
}

func (a Auth) enabled() bool { return a.Token != "" }

// accessCookieName 是携带访问凭证的 Cookie 名。Cookie 的值直接等于 Auth.Token，
// 服务端不做任何会话状态（design D2）。
const accessCookieName = "access_token"

// GenerateToken 生成启用认证时的访问凭证：crypto/rand 取 32 字节，经
// base64.RawURLEncoding 编码为 URL 安全字符串，可直接放进 ?token=（design D4）。
func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// tokenMatches 用常量时间比较判定凭证是否相符（design D4）。期望凭证为空
// （未启用认证）时一律不匹配，调用方无需另行判空。
func tokenMatches(want, got string) bool {
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

// authMiddleware 在启用认证时把认证门包在 next 外层：所有请求先过门，再进现有 mux，
// 因此现有端点内部零改动（design D5）。未启用认证（回环）时原样返回 next。
func authMiddleware(auth Auth, loginPage []byte, next http.Handler) http.Handler {
	if !auth.enabled() {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1) ?token=<有效>：种 Cookie 并 302 到去掉 token、保留其余参数的同一 URL（design D6）。
		if provided := r.URL.Query().Get("token"); provided != "" && tokenMatches(auth.Token, provided) {
			http.SetCookie(w, accessCookie(auth.Token))
			http.Redirect(w, r, stripToken(r.URL), http.StatusFound)
			return
		}

		// 2) 携带有效 Cookie：放行。
		if c, err := r.Cookie(accessCookieName); err == nil && tokenMatches(auth.Token, c.Value) {
			next.ServeHTTP(w, r)
			return
		}

		// 3) 拒绝：/api/ 前缀走 JSON 错误信封，其余（页面、静态资源）走自包含登录页。
		// 按路径前缀区分失败表现，对应 authentication: Authentication failure reporting。
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, fail(codeUnauthorized, "未认证：需要有效访问凭证"))
			return
		}
		writeLoginPage(w, loginPage)
	})
}

// accessCookie 构造携带凭证的会话 Cookie（design D6）：HttpOnly 降低 XSS 泄露面，
// SameSite=Strict 顶住 CSRF，Path=/ 全站适用；不设 Secure（纯 HTTP，设了浏览器不保存），
// 不设 Max-Age（会话 Cookie，关浏览器即失效）。
func accessCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     accessCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

// stripToken 返回去掉 token 查询参数、保留其余参数的同一 URL（design D6）。
// 以解析后的形式重建：token 不出现在重定向目标里，浏览位置因而不再包含凭证。
func stripToken(u *url.URL) string {
	q := u.Query()
	q.Del("token")
	clean := *u
	clean.RawQuery = q.Encode()
	return clean.String()
}

// writeLoginPage 以 401 输出自包含登录页（design D7）：页面不引用任何外部资源，
// 否则那些请求同样未认证、拿不到资源。
func writeLoginPage(w http.ResponseWriter, page []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(page)
}

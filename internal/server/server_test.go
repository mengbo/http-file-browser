package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mengbo/http-file-browser/web"
)

func TestServiceHealthIsQueried(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	NewAPIHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q，期望 JSON", contentType)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v（body = %q）", err, recorder.Body.String())
	}
	if body.Status != "ok" {
		t.Errorf("status = %q，期望 %q", body.Status, "ok")
	}
}

func TestUnknownAPIEndpointReturnsJSONError(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/nope", nil)

	NewAPIHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("状态码 = %d，期望 %d", recorder.Code, http.StatusNotFound)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q，期望 JSON", contentType)
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是可反序列化的 JSON：%v（body = %q）", err, recorder.Body.String())
	}
	if body.Error == "" {
		t.Error("错误响应缺少机器可读的错误标识（error 字段）")
	}
	if !strings.Contains(body.Error, "/api/nope") {
		t.Errorf("error = %q，期望说明未找到的路径", body.Error)
	}
}

func TestAPIPrefixTakesPrecedenceOverFileService(t *testing.T) {
	handler := NewHandler(web.FS)

	for _, path := range []string{"/api/nope", "/api/", "/api/unknown/deep"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q，期望 JSON（%s 落到了文件服务上）", contentType, path)
			}
			if strings.Contains(recorder.Body.String(), "<html") {
				t.Errorf("响应体是 HTML 页面，期望 JSON 错误信封：%q", recorder.Body.String())
			}
		})
	}
}

func TestRootPathReturnsFrontendPage(t *testing.T) {
	recorder := httptest.NewRecorder()

	NewHandler(web.FS).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q，期望 HTML", contentType)
	}
	if !strings.Contains(recorder.Body.String(), "/app.js") {
		t.Errorf("根路径响应未引用 app.js：%q", recorder.Body.String())
	}
}

func TestFrontendStaticAssetsAreServedWithMatchingContentType(t *testing.T) {
	handler := NewHandler(web.FS)

	for path, wantType := range map[string]string{
		"/app.js":    "text/javascript",
		"/style.css": "text/css",
	} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

			if recorder.Code != http.StatusOK {
				t.Errorf("状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
			}
			if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, wantType) {
				t.Errorf("Content-Type = %q，期望以 %q 开头", contentType, wantType)
			}
			if recorder.Body.Len() == 0 {
				t.Error("静态资源响应体为空")
			}
		})
	}
}

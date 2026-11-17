//go:build integration

package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/De3trvctions/Stallink/internal/auth"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
)

func TestAuthenticationAndRoleLifecycle(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := isolatedSchemaPool(t, ctx, dsn)
	if err := data.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	store := data.NewShopStore(pool)
	handler := server.NewHTTP(":0", "auth-test", &service.API{DB: store})

	prefix := "v" + strconv.FormatInt(time.Now().UnixNano(), 36)
	password := "test-password-123"
	platformLogin := prefix + "-platform"
	if err := data.EnsurePlatformAdmin(ctx, store, platformLogin, password); err != nil {
		t.Fatal(err)
	}

	platformCookie := login(t, handler, platformLogin, password)
	shopSlug := prefix + "-shop"
	ownerLogin := prefix + "-owner"
	opened := requestJSON(t, handler, http.MethodPost, "/api/platform/shops", map[string]string{
		"name": "Login Test Shop", "slug": shopSlug, "notice": "", "adminLogin": ownerLogin, "adminPassword": password,
	}, platformCookie)
	if opened.Code != http.StatusCreated {
		t.Fatalf("open shop: status %d, body %s", opened.Code, opened.Body.String())
	}
	ownerCookie := login(t, handler, ownerLogin, password)

	salesLogin := prefix + "-sales"
	salesCreated := requestJSON(t, handler, http.MethodPost, "/api/admin/sales", map[string]string{
		"loginName": salesLogin, "password": password,
	}, ownerCookie)
	if salesCreated.Code != http.StatusCreated {
		t.Fatalf("create sales account: status %d, body %s", salesCreated.Code, salesCreated.Body.String())
	}
	var sales struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(salesCreated.Body.Bytes(), &sales); err != nil || sales.ID == "" {
		t.Fatalf("decode sales account id: body %s, err %v", salesCreated.Body.String(), err)
	}

	customerLogin := prefix + "-customer"
	registered := requestJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{
		"loginName": customerLogin, "password": password, "displayName": "Test Customer", "contact": "test contact",
	}, nil)
	if registered.Code != http.StatusOK || responseRole(t, registered) != "customer" {
		t.Fatalf("register customer: status %d, body %s", registered.Code, registered.Body.String())
	}
	customerCookie := responseCookie(t, registered)
	registeredMe := requestJSON(t, handler, http.MethodGet, "/api/auth/me", nil, customerCookie)
	if registeredMe.Code != http.StatusOK || responseRole(t, registeredMe) != "customer" {
		t.Fatalf("registered customer session: status %d, body %s", registeredMe.Code, registeredMe.Body.String())
	}
	var customer struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(registered.Body.Bytes(), &customer); err != nil || customer.ID == "" {
		t.Fatalf("decode customer id: body %s, err %v", registered.Body.String(), err)
	}

	if duplicate := requestJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{
		"loginName": customerLogin, "password": password,
	}, nil); duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate registration: status %d, body %s", duplicate.Code, duplicate.Body.String())
	}
	if invalid := requestJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{
		"loginName": prefix + "-short", "password": "short",
	}, nil); invalid.Code != http.StatusBadRequest {
		t.Fatalf("short password registration: status %d, body %s", invalid.Code, invalid.Body.String())
	}

	accounts := []struct {
		login string
		role  string
	}{
		{login: customerLogin, role: "customer"},
		{login: salesLogin, role: "sales"},
		{login: ownerLogin, role: "shop_admin"},
		{login: platformLogin, role: "platform_admin"},
	}
	cookies := make(map[string]*http.Cookie, len(accounts))
	for _, account := range accounts {
		t.Run("login/"+account.role, func(t *testing.T) {
			response := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{
				"loginName": account.login, "password": password,
			}, nil)
			if response.Code != http.StatusOK || responseRole(t, response) != account.role {
				t.Fatalf("login %s: status %d, body %s", account.role, response.Code, response.Body.String())
			}
			cookie := responseCookie(t, response)
			if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.MaxAge != 14*24*3600 {
				t.Fatalf("session cookie flags: %+v", cookie)
			}
			cookies[account.role] = cookie

			me := requestJSON(t, handler, http.MethodGet, "/api/auth/me", nil, cookie)
			if me.Code != http.StatusOK || responseRole(t, me) != account.role {
				t.Fatalf("me %s: status %d, body %s", account.role, me.Code, me.Body.String())
			}
		})
	}

	if noSession := requestJSON(t, handler, http.MethodGet, "/api/auth/me", nil, nil); noSession.Code != http.StatusUnauthorized {
		t.Fatalf("me without session: status %d, body %s", noSession.Code, noSession.Body.String())
	}
	for _, credentials := range []struct {
		name     string
		login    string
		password string
	}{
		{name: "unknown account", login: prefix + "-missing", password: password},
		{name: "wrong password", login: customerLogin, password: "wrong-password"},
	} {
		t.Run("login failure/"+credentials.name, func(t *testing.T) {
			response := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{
				"loginName": credentials.login, "password": credentials.password,
			}, nil)
			if response.Code != http.StatusUnauthorized || !bytes.Contains(response.Body.Bytes(), []byte(`"error":"login_failed"`)) {
				t.Fatalf("login failure %s: status %d, body %s", credentials.name, response.Code, response.Body.String())
			}
			if len(response.Result().Cookies()) != 0 {
				t.Fatalf("failed login %s set a session cookie", credentials.name)
			}
		})
	}

	profile := requestJSON(t, handler, http.MethodPatch, "/api/me", map[string]string{
		"displayName": "Updated Customer", "contact": "updated contact",
	}, cookies["customer"])
	if profile.Code != http.StatusOK {
		t.Fatalf("update customer profile: status %d, body %s", profile.Code, profile.Body.String())
	}

	for _, access := range []struct {
		name   string
		role   string
		path   string
		status int
	}{
		{name: "customer denied staff", role: "customer", path: "/api/staff/events", status: http.StatusForbidden},
		{name: "sales denied admin", role: "sales", path: "/api/admin/sales", status: http.StatusForbidden},
		{name: "sales allowed staff", role: "sales", path: "/api/staff/events", status: http.StatusOK},
		{name: "shop admin allowed staff", role: "shop_admin", path: "/api/staff/events", status: http.StatusOK},
		{name: "shop admin denied platform", role: "shop_admin", path: "/api/platform/shops", status: http.StatusForbidden},
		{name: "platform denied staff", role: "platform_admin", path: "/api/staff/events", status: http.StatusForbidden},
		{name: "platform allowed platform", role: "platform_admin", path: "/api/platform/shops", status: http.StatusOK},
	} {
		t.Run("authorization/"+access.name, func(t *testing.T) {
			response := requestJSON(t, handler, http.MethodGet, access.path, nil, cookies[access.role])
			if response.Code != access.status {
				t.Fatalf("%s: status %d, body %s; want %d", access.name, response.Code, response.Body.String(), access.status)
			}
		})
	}

	disabled := requestJSON(t, handler, http.MethodPost, "/api/admin/sales/"+sales.ID+"/disable", map[string]bool{}, ownerCookie)
	if disabled.Code != http.StatusOK {
		t.Fatalf("disable sales: status %d, body %s", disabled.Code, disabled.Body.String())
	}
	if disabledLogin := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{
		"loginName": salesLogin, "password": password,
	}, nil); disabledLogin.Code != http.StatusUnauthorized {
		t.Fatalf("disabled sales login: status %d, body %s", disabledLogin.Code, disabledLogin.Body.String())
	}
	if disabledMe := requestJSON(t, handler, http.MethodGet, "/api/auth/me", nil, cookies["sales"]); disabledMe.Code != http.StatusUnauthorized {
		t.Fatalf("disabled sales existing session: status %d, body %s", disabledMe.Code, disabledMe.Body.String())
	}

	enabled := requestJSON(t, handler, http.MethodPost, "/api/admin/sales/"+sales.ID+"/enable", map[string]bool{}, ownerCookie)
	if enabled.Code != http.StatusOK {
		t.Fatalf("enable sales: status %d, body %s", enabled.Code, enabled.Body.String())
	}
	newPassword := "new-test-password-123"
	reset := requestJSON(t, handler, http.MethodPost, "/api/admin/sales/"+sales.ID+"/password", map[string]string{
		"password": newPassword,
	}, ownerCookie)
	if reset.Code != http.StatusOK {
		t.Fatalf("reset sales password: status %d, body %s", reset.Code, reset.Body.String())
	}
	if oldPassword := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{
		"loginName": salesLogin, "password": password,
	}, nil); oldPassword.Code != http.StatusUnauthorized {
		t.Fatalf("old sales password: status %d, body %s", oldPassword.Code, oldPassword.Body.String())
	}
	if newPasswordLogin := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{
		"loginName": salesLogin, "password": newPassword,
	}, nil); newPasswordLogin.Code != http.StatusOK || responseRole(t, newPasswordLogin) != "sales" {
		t.Fatalf("new sales password: status %d, body %s", newPasswordLogin.Code, newPasswordLogin.Body.String())
	}

	logout := requestJSON(t, handler, http.MethodPost, "/api/auth/logout", map[string]bool{}, cookies["customer"])
	if logout.Code != http.StatusOK {
		t.Fatalf("logout: status %d, body %s", logout.Code, logout.Body.String())
	}
	cleared := responseCookie(t, logout)
	if cleared.MaxAge != -1 || cleared.Value != "" {
		t.Fatalf("logout did not clear session cookie: %+v", cleared)
	}
	if afterLogout := requestJSON(t, handler, http.MethodGet, "/api/auth/me", nil, cookies["customer"]); afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("session after logout: status %d, body %s", afterLogout.Code, afterLogout.Body.String())
	}

	// An expired server-side record cannot be replayed even with a syntactically valid token.
	expiredToken, expiredHash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	customerID, err := data.ParseID(customer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.InsertSession(ctx, customerID, expiredHash, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	expired := &http.Cookie{Name: "stallink_session", Value: expiredToken}
	if response := requestJSON(t, handler, http.MethodGet, "/api/auth/me", nil, expired); response.Code != http.StatusUnauthorized {
		t.Fatalf("expired session: status %d, body %s", response.Code, response.Body.String())
	}
}

func login(t *testing.T, handler http.Handler, loginName, password string) *http.Cookie {
	t.Helper()
	response := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{
		"loginName": loginName, "password": password,
	}, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("login %q: status %d, body %s", loginName, response.Code, response.Body.String())
	}
	return responseCookie(t, response)
}

func requestJSON(t *testing.T, handler http.Handler, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func responseCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("response did not set a cookie: status %d, body %s", response.Code, response.Body.String())
	}
	for _, cookie := range cookies {
		if cookie.Name == "stallink_session" {
			return cookie
		}
	}
	t.Fatalf("response did not set stallink_session: %+v", cookies)
	return nil
}

func responseRole(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response role: %v; body %s", err, response.Body.String())
	}
	return body.Role
}

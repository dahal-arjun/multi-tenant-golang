// Command api_smoke_test runs HTTP smoke checks against the multi-tenant API (same coverage as scripts/api_smoke_test.py).
//
// Prerequisites: API running, DB migrated. Optional: ADMIN_EMAIL + ADMIN_PASSWORD for platform routes.
// Local-only tokens (invite_token, verification_token, reset_token) require API ENVIRONMENT=local.
//
// Usage:
//
//	go run ./scripts/api_smoke_test
//	go run ./scripts/api_smoke_test -base-url http://127.0.0.1:5000
//	ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=changeme go run ./scripts/api_smoke_test
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type apiClient struct {
	client  *http.Client
	baseURL string
}

func newAPIClient(baseURL string) (*apiClient, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if _, err := url.Parse(baseURL); err != nil {
		return nil, err
	}
	return &apiClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (c *apiClient) do(method, path string, query url.Values, body any, header http.Header) (int, []byte, error) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, reqURL, rdr)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, vv := range header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	return resp.StatusCode, raw, err
}

func decodeJSONMap(raw []byte) (map[string]interface{}, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func jwtPayloadUnverified(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("not a JWT")
	}
	payloadSeg := parts[1]
	raw, err := base64.RawURLEncoding.DecodeString(payloadSeg)
	if err != nil {
		if n := len(payloadSeg) % 4; n != 0 {
			payloadSeg += strings.Repeat("=", 4-n)
		}
		raw, err = base64.URLEncoding.DecodeString(payloadSeg)
		if err != nil {
			return nil, err
		}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func claimString(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func claimInt(m map[string]interface{}, key string) (int, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return int(x), true
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func dataField(body map[string]interface{}) (map[string]interface{}, error) {
	d, ok := body["data"]
	if !ok {
		return nil, fmt.Errorf("expected JSON with \"data\" key, got %#v", body)
	}
	dm, ok := d.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected data object, got %T", d)
	}
	return dm, nil
}

func fail(step, detail string) {
	fmt.Fprintf(os.Stderr, "FAIL: %s\n      %s\n", step, detail)
	os.Exit(1)
}

func ok(step string) {
	fmt.Printf("ok   %s\n", step)
}

func expectStatus(code, want int, step string, body any) {
	if code != want {
		fail(step, fmt.Sprintf("HTTP %d (expected %d): %v", code, want, body))
	}
}

func main() {
	baseFlag := flag.String("base-url", "", "API origin (default: env API_BASE_URL or http://127.0.0.1:5000)")
	flag.Parse()

	base := strings.TrimSpace(*baseFlag)
	if base == "" {
		base = strings.TrimSpace(os.Getenv("API_BASE_URL"))
	}
	if base == "" {
		base = "http://127.0.0.1:5000"
	}

	client, err := newAPIClient(base)
	if err != nil {
		fail("config", err.Error())
	}

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	const password = "smokeTest!234"
	const password2 = "smokeTest!567"
	emailOwner := fmt.Sprintf("smoke-owner-%s@example.com", tag)
	emailInvitee := fmt.Sprintf("smoke-invite-%s@example.com", tag)
	emailSignup := fmt.Sprintf("smoke-signup-%s@example.com", tag)
	emailResend := fmt.Sprintf("smoke-resend-%s@example.com", tag)
	emailForgot := fmt.Sprintf("smoke-forgot-%s@example.com", tag)

	authHdr := func(token string) http.Header {
		h := make(http.Header)
		h.Set("Authorization", "Bearer "+token)
		return h
	}

	// --- Public health ---
	code, raw, err := client.do(http.MethodGet, "/health-check", nil, nil, nil)
	if err != nil {
		fail("GET /health-check", err.Error())
	}
	hb, _ := decodeJSONMap(raw)
	expectStatus(code, 200, "GET /health-check", hb)
	ok("GET /health-check")

	// --- Register ---
	code, raw, err = client.do(http.MethodPost, "/api/auth/register", nil, map[string]interface{}{
		"email":       emailOwner,
		"password":    password,
		"tenant_name": fmt.Sprintf("Smoke Org %s", tag),
	}, nil)
	if err != nil {
		fail("POST /api/auth/register", err.Error())
	}
	regBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/register", err.Error())
	}
	expectStatus(code, 201, "POST /api/auth/register", regBody)
	reg, err := dataField(regBody)
	if err != nil {
		fail("POST /api/auth/register", err.Error())
	}
	access, _ := reg["access_token"].(string)
	refresh, _ := reg["refresh_token"].(string)
	if access == "" || refresh == "" {
		fail("POST /api/auth/register", "missing tokens")
	}
	claims, err := jwtPayloadUnverified(access)
	if err != nil {
		fail("register JWT", err.Error())
	}
	tenantID, okk := claimString(claims, "tid")
	if !okk {
		fail("register JWT", fmt.Sprintf("missing tid in claims: %#v", claims))
	}
	ok("POST /api/auth/register")

	// --- Me & user ---
	code, raw, err = client.do(http.MethodGet, "/api/auth/me", nil, nil, authHdr(access))
	if err != nil {
		fail("GET /api/auth/me", err.Error())
	}
	meBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("GET /api/auth/me", err.Error())
	}
	expectStatus(code, 200, "GET /api/auth/me", meBody)
	me, err := dataField(meBody)
	if err != nil {
		fail("GET /api/auth/me", err.Error())
	}
	userObj, okk := me["user"].(map[string]interface{})
	if !okk {
		fail("GET /api/auth/me", "user object")
	}
	userUUID, _ := userObj["uuid"].(string)
	if userUUID == "" {
		fail("GET /api/auth/me", "uuid")
	}
	if tid, _ := me["tenant_id"].(string); tid != tenantID {
		fail("GET /api/auth/me", "tenant_id mismatch")
	}
	ok("GET /api/auth/me")

	code, raw, err = client.do(http.MethodGet, "/api/user/"+url.PathEscape(userUUID), nil, nil, authHdr(access))
	if err != nil {
		fail("GET /api/user/{uuid}", err.Error())
	}
	uBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("GET /api/user/{uuid}", err.Error())
	}
	expectStatus(code, 200, "GET /api/user/{uuid}", uBody)
	ok("GET /api/user/{uuid}")

	code, raw, err = client.do(http.MethodPatch, "/api/auth/me", nil, map[string]interface{}{
		"first_name": "Smoke",
		"last_name":  "Tester",
	}, authHdr(access))
	if err != nil {
		fail("PATCH /api/auth/me", err.Error())
	}
	patchBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("PATCH /api/auth/me", err.Error())
	}
	expectStatus(code, 200, "PATCH /api/auth/me", patchBody)
	me2, err := dataField(patchBody)
	if err != nil {
		fail("PATCH /api/auth/me", err.Error())
	}
	u2, _ := me2["user"].(map[string]interface{})
	if u2["first_name"] != "Smoke" {
		fail("PATCH /api/auth/me", fmt.Sprintf("%v", me2))
	}
	ok("PATCH /api/auth/me")

	// --- Paginated lists ---
	q := url.Values{}
	q.Set("page", "1")
	q.Set("limit", "5")
	code, raw, err = client.do(http.MethodGet, "/api/widgets", q, nil, authHdr(access))
	if err != nil {
		fail("GET /api/widgets", err.Error())
	}
	wBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("GET /api/widgets", err.Error())
	}
	expectStatus(code, 200, "GET /api/widgets", wBody)
	pag, okk := wBody["pagination"].(map[string]interface{})
	if !okk {
		fail("GET /api/widgets", "missing pagination envelope")
	}
	for _, k := range []string{"page", "limit", "total", "total_pages", "has_next"} {
		if _, exists := pag[k]; !exists {
			fail("GET /api/widgets", "missing pagination."+k)
		}
	}
	ok("GET /api/widgets (paginated)")

	q2 := url.Values{}
	q2.Set("page", "1")
	q2.Set("limit", "10")
	code, raw, err = client.do(http.MethodGet, "/api/tenant-roles", q2, nil, authHdr(access))
	if err != nil {
		fail("GET /api/tenant-roles", err.Error())
	}
	trBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("GET /api/tenant-roles", err.Error())
	}
	expectStatus(code, 200, "GET /api/tenant-roles", trBody)
	if _, exists := trBody["pagination"]; !exists {
		fail("GET /api/tenant-roles", "missing pagination envelope")
	}
	ok("GET /api/tenant-roles (paginated)")

	// --- Tenant role CRUD ---
	code, raw, err = client.do(http.MethodPost, "/api/tenant-roles", nil, map[string]interface{}{
		"name":        fmt.Sprintf("Smoke Role %s", tag),
		"description": "api_smoke_test",
	}, authHdr(access))
	if err != nil {
		fail("POST /api/tenant-roles", err.Error())
	}
	roleCreateBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/tenant-roles", err.Error())
	}
	expectStatus(code, 201, "POST /api/tenant-roles", roleCreateBody)
	role, err := dataField(roleCreateBody)
	if err != nil {
		fail("POST /api/tenant-roles", err.Error())
	}
	roleID, _ := role["id"].(string)
	if roleID == "" {
		fail("POST /api/tenant-roles", "role id")
	}
	ok("POST /api/tenant-roles")

	code, raw, err = client.do(http.MethodPatch, "/api/tenant-roles/"+url.PathEscape(roleID), nil, map[string]interface{}{
		"name": fmt.Sprintf("Smoke Role %s (updated)", tag),
	}, authHdr(access))
	if err != nil {
		fail("PATCH /api/tenant-roles/{id}", err.Error())
	}
	patchRoleBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("PATCH /api/tenant-roles/{id}", err.Error())
	}
	expectStatus(code, 200, "PATCH /api/tenant-roles/{id}", patchRoleBody)
	ok("PATCH /api/tenant-roles/{id}")

	code, raw, err = client.do(http.MethodPut, "/api/tenant-roles/"+url.PathEscape(roleID)+"/permissions", nil, map[string]interface{}{
		"permissions": []string{"widgets.read", "widgets.write"},
	}, authHdr(access))
	if err != nil {
		fail("PUT /api/tenant-roles/{id}/permissions", err.Error())
	}
	putPermBody, _ := decodeJSONMap(raw)
	expectStatus(code, http.StatusNoContent, "PUT /api/tenant-roles/{id}/permissions", putPermBody)
	ok("PUT /api/tenant-roles/{id}/permissions")

	// --- Invite + accept + assign ---
	code, raw, err = client.do(http.MethodPost, "/api/invites", nil, map[string]interface{}{
		"email": emailInvitee,
		"role":  "member",
	}, authHdr(access))
	if err != nil {
		fail("POST /api/invites", err.Error())
	}
	invBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/invites", err.Error())
	}
	expectStatus(code, 201, "POST /api/invites", invBody)
	inv, err := dataField(invBody)
	if err != nil {
		fail("POST /api/invites", err.Error())
	}
	inviteToken, _ := inv["invite_token"].(string)
	if inviteToken == "" {
		fail("POST /api/invites", "invite_token missing (need ENVIRONMENT=local for token in response)")
	}
	ok("POST /api/invites")

	code, raw, err = client.do(http.MethodPost, "/api/auth/accept-invite", nil, map[string]interface{}{
		"token":    inviteToken,
		"password": password,
	}, nil)
	if err != nil {
		fail("POST /api/auth/accept-invite", err.Error())
	}
	accBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/accept-invite", err.Error())
	}
	expectStatus(code, 200, "POST /api/auth/accept-invite", accBody)
	invTok, err := dataField(accBody)
	if err != nil {
		fail("POST /api/auth/accept-invite", err.Error())
	}
	invAccess, _ := invTok["access_token"].(string)
	if invAccess == "" {
		fail("POST /api/auth/accept-invite", "access_token")
	}
	invClaims, err := jwtPayloadUnverified(invAccess)
	if err != nil {
		fail("POST /api/auth/accept-invite", err.Error())
	}
	inviteeID, okk := claimInt(invClaims, "idb")
	if !okk {
		fail("POST /api/auth/accept-invite", "idb in jwt")
	}
	ok("POST /api/auth/accept-invite")

	code, raw, err = client.do(http.MethodPatch, "/api/tenant-members/role", nil, map[string]interface{}{
		"user_id":        inviteeID,
		"tenant_role_id": roleID,
	}, authHdr(access))
	if err != nil {
		fail("PATCH /api/tenant-members/role (assign)", err.Error())
	}
	assignBody, _ := decodeJSONMap(raw)
	expectStatus(code, http.StatusNoContent, "PATCH /api/tenant-members/role (assign)", assignBody)
	ok("PATCH /api/tenant-members/role (assign)")

	code, raw, err = client.do(http.MethodPatch, "/api/tenant-members/role", nil, map[string]interface{}{
		"user_id":        inviteeID,
		"tenant_role_id": nil,
	}, authHdr(access))
	if err != nil {
		fail("PATCH /api/tenant-members/role (clear)", err.Error())
	}
	clearBody, _ := decodeJSONMap(raw)
	expectStatus(code, http.StatusNoContent, "PATCH /api/tenant-members/role (clear)", clearBody)
	ok("PATCH /api/tenant-members/role (clear)")

	code, raw, err = client.do(http.MethodDelete, "/api/tenant-roles/"+url.PathEscape(roleID), nil, nil, authHdr(access))
	if err != nil {
		fail("DELETE /api/tenant-roles/{id}", err.Error())
	}
	delBody, _ := decodeJSONMap(raw)
	expectStatus(code, http.StatusNoContent, "DELETE /api/tenant-roles/{id}", delBody)
	ok("DELETE /api/tenant-roles/{id}")

	// --- Refresh + logout ---
	code, raw, err = client.do(http.MethodPost, "/api/auth/refresh", nil, map[string]interface{}{
		"refresh_token": refresh,
	}, nil)
	if err != nil {
		fail("POST /api/auth/refresh", err.Error())
	}
	refBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/refresh", err.Error())
	}
	expectStatus(code, 200, "POST /api/auth/refresh", refBody)
	refData, err := dataField(refBody)
	if err != nil {
		fail("POST /api/auth/refresh", err.Error())
	}
	refresh, _ = refData["refresh_token"].(string)
	access, _ = refData["access_token"].(string)
	if access == "" || refresh == "" {
		fail("POST /api/auth/refresh", "tokens")
	}
	ok("POST /api/auth/refresh")

	code, raw, err = client.do(http.MethodPost, "/api/auth/logout", nil, map[string]interface{}{
		"refresh_token": refresh,
	}, authHdr(access))
	if err != nil {
		fail("POST /api/auth/logout", err.Error())
	}
	loBody, _ := decodeJSONMap(raw)
	expectStatus(code, http.StatusNoContent, "POST /api/auth/logout", loBody)
	ok("POST /api/auth/logout")

	// --- Login + tenant-session ---
	code, raw, err = client.do(http.MethodPost, "/api/auth/login", nil, map[string]interface{}{
		"email":    emailOwner,
		"password": password,
	}, nil)
	if err != nil {
		fail("POST /api/auth/login", err.Error())
	}
	logBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/login", err.Error())
	}
	expectStatus(code, 200, "POST /api/auth/login", logBody)
	disc, err := dataField(logBody)
	if err != nil {
		fail("POST /api/auth/login", err.Error())
	}
	pick, _ := disc["pick_tenant_token"].(string)
	tenants, _ := disc["tenants"].([]interface{})
	if pick == "" || len(tenants) == 0 {
		fail("POST /api/auth/login", "expected pick token and tenants")
	}
	t0, _ := tenants[0].(map[string]interface{})
	tid0, _ := t0["tenant_id"].(string)
	if tid0 == "" {
		fail("POST /api/auth/login", "tenant_id")
	}
	ok("POST /api/auth/login")

	code, raw, err = client.do(http.MethodPost, "/api/auth/tenant-session", nil, map[string]interface{}{
		"tenant_id": tid0,
	}, authHdr(pick))
	if err != nil {
		fail("POST /api/auth/tenant-session", err.Error())
	}
	tsBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/tenant-session", err.Error())
	}
	expectStatus(code, 200, "POST /api/auth/tenant-session", tsBody)
	tsData, err := dataField(tsBody)
	if err != nil {
		fail("POST /api/auth/tenant-session", err.Error())
	}
	access, _ = tsData["access_token"].(string)
	refresh, _ = tsData["refresh_token"].(string)
	if access == "" {
		fail("POST /api/auth/tenant-session", "access_token")
	}
	ok("POST /api/auth/tenant-session")

	// --- Change password ---
	code, raw, err = client.do(http.MethodPost, "/api/auth/change-password", nil, map[string]interface{}{
		"current_password": password,
		"new_password":     password2,
	}, authHdr(access))
	if err != nil {
		fail("POST /api/auth/change-password", err.Error())
	}
	cpBody, _ := decodeJSONMap(raw)
	expectStatus(code, http.StatusNoContent, "POST /api/auth/change-password", cpBody)
	ok("POST /api/auth/change-password")

	code, raw, err = client.do(http.MethodPost, "/api/auth/login", nil, map[string]interface{}{
		"email":    emailOwner,
		"password": password2,
	}, nil)
	if err != nil {
		fail("POST /api/auth/login (after password change)", err.Error())
	}
	log2Body, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/login (after password change)", err.Error())
	}
	expectStatus(code, 200, "POST /api/auth/login (after password change)", log2Body)
	ok("POST /api/auth/login (after password change)")

	// --- Resend verification ---
	code, raw, err = client.do(http.MethodPost, "/api/auth/signup", nil, map[string]interface{}{
		"email":    emailResend,
		"password": password,
	}, nil)
	if err != nil {
		fail("POST /api/auth/signup (resend user)", err.Error())
	}
	su0, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/signup (resend user)", err.Error())
	}
	expectStatus(code, 201, "POST /api/auth/signup (resend user)", su0)
	ok("POST /api/auth/signup (resend user)")

	code, raw, err = client.do(http.MethodPost, "/api/auth/resend-verification", nil, map[string]interface{}{
		"email": emailResend,
	}, nil)
	if err != nil {
		fail("POST /api/auth/resend-verification", err.Error())
	}
	rvBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/resend-verification", err.Error())
	}
	expectStatus(code, 200, "POST /api/auth/resend-verification", rvBody)
	ok("POST /api/auth/resend-verification")

	// --- Signup + verify + create-tenant ---
	code, raw, err = client.do(http.MethodPost, "/api/auth/signup", nil, map[string]interface{}{
		"email":    emailSignup,
		"password": password,
	}, nil)
	if err != nil {
		fail("POST /api/auth/signup", err.Error())
	}
	suBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/signup", err.Error())
	}
	expectStatus(code, 201, "POST /api/auth/signup", suBody)
	su, err := dataField(suBody)
	if err != nil {
		fail("POST /api/auth/signup", err.Error())
	}
	vtok, _ := su["verification_token"].(string)
	if vtok == "" {
		fmt.Println("skip POST /api/auth/verify-email & create-tenant: no verification_token (set API to ENVIRONMENT=local)")
	} else {
		code, raw, err = client.do(http.MethodPost, "/api/auth/verify-email", nil, map[string]interface{}{
			"token": vtok,
		}, nil)
		if err != nil {
			fail("POST /api/auth/verify-email", err.Error())
		}
		veBody, _ := decodeJSONMap(raw)
		expectStatus(code, http.StatusNoContent, "POST /api/auth/verify-email", veBody)
		ok("POST /api/auth/verify-email")

		code, raw, err = client.do(http.MethodPost, "/api/auth/login", nil, map[string]interface{}{
			"email":    emailSignup,
			"password": password,
		}, nil)
		if err != nil {
			fail("POST /api/auth/login (signup user)", err.Error())
		}
		slBody, err := decodeJSONMap(raw)
		if err != nil {
			fail("POST /api/auth/login (signup user)", err.Error())
		}
		expectStatus(code, 200, "POST /api/auth/login (signup user)", slBody)
		disc2, err := dataField(slBody)
		if err != nil {
			fail("POST /api/auth/login (signup user)", err.Error())
		}
		pick2, _ := disc2["pick_tenant_token"].(string)
		if pick2 == "" {
			fail("POST /api/auth/login (signup user)", "pick token")
		}

		code, raw, err = client.do(http.MethodPost, "/api/auth/create-tenant", nil, map[string]interface{}{
			"tenant_name": fmt.Sprintf("Signup Co %s", tag),
		}, authHdr(pick2))
		if err != nil {
			fail("POST /api/auth/create-tenant", err.Error())
		}
		ctBody, err := decodeJSONMap(raw)
		if err != nil {
			fail("POST /api/auth/create-tenant", err.Error())
		}
		expectStatus(code, 201, "POST /api/auth/create-tenant", ctBody)
		ok("POST /api/auth/create-tenant")
	}

	// --- Forgot + reset ---
	code, raw, err = client.do(http.MethodPost, "/api/auth/register", nil, map[string]interface{}{
		"email":       emailForgot,
		"password":    password,
		"tenant_name": fmt.Sprintf("Forgot Org %s", tag),
	}, nil)
	if err != nil {
		fail("POST /api/auth/register (forgot-flow user)", err.Error())
	}
	fgReg, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/register (forgot-flow user)", err.Error())
	}
	expectStatus(code, 201, "POST /api/auth/register (forgot-flow user)", fgReg)
	ok("POST /api/auth/register (forgot-flow user)")

	code, raw, err = client.do(http.MethodPost, "/api/auth/forgot-password", nil, map[string]interface{}{
		"email": emailForgot,
	}, nil)
	if err != nil {
		fail("POST /api/auth/forgot-password", err.Error())
	}
	fpBody, err := decodeJSONMap(raw)
	if err != nil {
		fail("POST /api/auth/forgot-password", err.Error())
	}
	expectStatus(code, 200, "POST /api/auth/forgot-password", fpBody)
	fp, err := dataField(fpBody)
	if err != nil {
		fail("POST /api/auth/forgot-password", err.Error())
	}
	resetTok, _ := fp["reset_token"].(string)
	if resetTok == "" {
		fmt.Println("skip POST /api/auth/reset-password: no reset_token (need ENVIRONMENT=local)")
	} else {
		code, raw, err = client.do(http.MethodPost, "/api/auth/reset-password", nil, map[string]interface{}{
			"token":        resetTok,
			"new_password": password2,
		}, nil)
		if err != nil {
			fail("POST /api/auth/reset-password", err.Error())
		}
		rsBody, _ := decodeJSONMap(raw)
		expectStatus(code, http.StatusNoContent, "POST /api/auth/reset-password", rsBody)
		ok("POST /api/auth/reset-password")
	}

	// --- Platform (optional) ---
	adminEmail := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	adminPass := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	if adminEmail == "" || adminPass == "" {
		fmt.Println("skip platform routes: set ADMIN_EMAIL and ADMIN_PASSWORD")
	} else {
		code, raw, err = client.do(http.MethodPost, "/api/auth/login", nil, map[string]interface{}{
			"email":    adminEmail,
			"password": adminPass,
		}, nil)
		if err != nil {
			fail("POST /api/auth/login (admin)", err.Error())
		}
		adBody, err := decodeJSONMap(raw)
		if err != nil {
			fail("POST /api/auth/login (admin)", err.Error())
		}
		expectStatus(code, 200, "POST /api/auth/login (admin)", adBody)
		adm, err := dataField(adBody)
		if err != nil {
			fail("POST /api/auth/login (admin)", err.Error())
		}
		plat, _ := adm["platform_access_token"].(string)
		if plat == "" {
			fail("platform", "admin login had no platform_access_token (user must be system_manager/admin seed)")
		}
		code, raw, err = client.do(http.MethodGet, "/api/platform/health", nil, nil, authHdr(plat))
		if err != nil {
			fail("GET /api/platform/health", err.Error())
		}
		phBody, err := decodeJSONMap(raw)
		if err != nil {
			fail("GET /api/platform/health", err.Error())
		}
		expectStatus(code, 200, "GET /api/platform/health", phBody)
		ok("GET /api/platform/health")

		code, raw, err = client.do(http.MethodGet, "/api/platform/tenants/summary", nil, nil, authHdr(plat))
		if err != nil {
			fail("GET /api/platform/tenants/summary", err.Error())
		}
	sumBody, err := decodeJSONMap(raw)
		if err != nil {
			fail("GET /api/platform/tenants/summary", err.Error())
		}
		expectStatus(code, 200, "GET /api/platform/tenants/summary", sumBody)
		ok("GET /api/platform/tenants/summary")
	}

	fmt.Println("\nAll executed checks passed.")
}

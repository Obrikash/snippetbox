package main

import (
	"github.com/obrikash/snippetbox/internal/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestPing(t *testing.T) {
	app := newTestApplication(t)

	ts := newTestServer(t, app.routes())
	defer ts.Close()

	code, _, body := ts.get(t, "/ping")

	assert.Equal(t, code, http.StatusOK)
	assert.Equal(t, body, "OK")
}

func TestSnippetView(t *testing.T) {
	app := newTestApplication(t)

	ts := newTestServer(t, app.routes())
	defer ts.Close()

	tests := []struct {
		name     string
		urlPath  string
		wantCode int
		wantBody string
	}{
		{name: "Valid ID", urlPath: "/snippet/view/1", wantCode: http.StatusOK, wantBody: "An old silent pond..."},
		{name: "Non-existent ID", urlPath: "/snippet/view/2", wantCode: http.StatusNotFound},
		{name: "Negative ID", urlPath: "/snippet/view/-1", wantCode: http.StatusNotFound},
		{name: "Decimal ID", urlPath: "/snippet/view/1.23", wantCode: http.StatusNotFound},
		{name: "String ID", urlPath: "/snippet/view/foo", wantCode: http.StatusNotFound},
		{name: "Empty ID", urlPath: "/snippet/view/", wantCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, body := ts.get(t, tt.urlPath)

			assert.Equal(t, code, tt.wantCode)

			if tt.wantBody != "" {
				assert.StringContains(t, body, tt.wantBody)
			}
		})
	}
}

func TestSnippetCreate(t *testing.T) {
    app := newTestApplication(t)

	ts := newTestServer(t, app.routes())
	defer ts.Close()

	t.Run("Unauthenticated", func(t *testing.T) {
		code, header, _ := ts.get(t, "/snippet/create")

		assert.Equal(t, code, http.StatusSeeOther)
		assert.Equal(t, header.Get("Location"), "/user/login")
	})

	t.Run("Authenticated", func(t *testing.T) {
        _, _, body := ts.get(t, "/user/login")
        validToken := extractCSRFToken(t, body)

        form := url.Values{}
        form.Add("email", "alice@example.com")
        form.Add("password", "pa$$word")
        form.Add("csrf_token", validToken)
        ts.postForm(t, "/user/login", form)

        code, _, body := ts.get(t, "/snippet/create")
        assert.Equal(t, code, http.StatusOK)
        assert.StringContains(t, body, "<form action='/snippet/create' method='POST'>")
	})
}

func TestUserSignup(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	_, _, body := ts.get(t, "/user/signup")
	validToken := extractCSRFToken(t, body)

	const (
		validName     = "bob"
		validPassword = "validPa$$word"
		validEmail    = "bob@example.com"
		formTag       = "<form action='/user/signup' method='POST' novalidate>"
	)

	tests := []struct {
		name         string
		userName     string
		userEmail    string
		userPassword string
		csrfToken    string
		wantCode     int
		wantFormTag  string
	}{
		{name: "Valid Submission", userName: validName, userEmail: validEmail, userPassword: validPassword, csrfToken: validToken, wantCode: http.StatusSeeOther},
		{name: "Invalid CSRF Token", userName: validName, userEmail: validEmail, userPassword: validPassword, csrfToken: "wrongToken", wantCode: http.StatusBadRequest},
		{name: "Empty name", userName: "", userEmail: validEmail, userPassword: validPassword, csrfToken: validToken, wantCode: http.StatusUnprocessableEntity, wantFormTag: formTag},
		{name: "Empty email",
			userName:     validName,
			userEmail:    "",
			userPassword: validPassword,
			csrfToken:    validToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag},
		{name: "Empty password",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: "",
			csrfToken:    validToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag},
		{name: "Invalid email",
			userName:     validName,
			userEmail:    "bob@example.",
			userPassword: validPassword,
			csrfToken:    validToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag},
		{name: "Short password",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: "pa$$",
			csrfToken:    validToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag},
		{name: "Short password",
			userName:     validName,
			userEmail:    validEmail,
			userPassword: "pa$$",
			csrfToken:    validToken,
			wantCode:     http.StatusUnprocessableEntity,
			wantFormTag:  formTag},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            form := url.Values{}
            form.Add("name", tt.userName)
            form.Add("email", tt.userEmail)
            form.Add("password", tt.userPassword)
            form.Add("csrf_token", tt.csrfToken)

            code, _, body := ts.postForm(t, "/user/signup", form)

            assert.Equal(t, code, tt.wantCode)

            if tt.wantFormTag != "" {
                assert.StringContains(t, body, tt.wantFormTag)
            }
        })
    }

}

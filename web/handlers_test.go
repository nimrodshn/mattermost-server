// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package web

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mattermost/mattermost-server/app"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handlerForHTTPErrors(c *Context, w http.ResponseWriter, r *http.Request) {
	c.Err = model.NewAppError("loginWithSaml", "api.user.saml.not_available.app_error", nil, "", http.StatusFound)
}

func TestHandlerServeHTTPErrors(t *testing.T) {
	permConfig, err := os.Open(fileutils.FindConfigFile("config.json"))
	if err != nil {
		panic(err)
	}
	defer permConfig.Close()
	tempConfig, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(tempConfig, permConfig)
	tempConfig.Close()
	if err != nil {
		panic(err)
	}

	options := []app.Option{app.ConfigFile(tempConfig.Name(), false)}
	options = append(options, app.StoreOverride(mainHelper.Store))

	s, err := app.NewServer(options...)
	require.Nil(t, err)
	defer s.Shutdown()

	web := New(s, s.AppOptions, s.Router)
	if err != nil {
		panic(err)
	}
	handler := web.NewHandler(handlerForHTTPErrors)

	var flagtests = []struct {
		name     string
		url      string
		mobile   bool
		redirect bool
	}{
		{"redirect on desktop non-api endpoint", "/login/sso/saml", false, true},
		{"not redirect on desktop api endpoint", "/api/v4/test", false, false},
		{"not redirect on mobile non-api endpoint", "/login/sso/saml", true, false},
		{"not redirect on mobile api endpoint", "/api/v4/test", true, false},
	}

	for _, tt := range flagtests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", tt.url, nil)
			if tt.mobile {
				request.Header.Add("X-Mobile-App", "mattermost")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if tt.redirect {
				assert.Equal(t, response.Code, http.StatusFound)
			} else {
				assert.NotContains(t, response.Body.String(), "/error?message=")
			}
		})
	}
}

func handlerForHTTPSecureTransport(c *Context, w http.ResponseWriter, r *http.Request) {
}

func TestHandlerServeHTTPSecureTransport(t *testing.T) {
	permConfig, err := os.Open(fileutils.FindConfigFile("config.json"))
	if err != nil {
		panic(err)
	}
	defer permConfig.Close()
	tempConfig, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(tempConfig, permConfig)
	tempConfig.Close()
	if err != nil {
		panic(err)
	}

	options := []app.Option{app.ConfigFile(tempConfig.Name(), false)}
	options = append(options, app.StoreOverride(mainHelper.Store))

	s, err := app.NewServer(options...)
	require.Nil(t, err)
	defer s.Shutdown()

	a := s.FakeApp()

	a.UpdateConfig(func(config *model.Config) {
		*config.ServiceSettings.TLSStrictTransport = true
		*config.ServiceSettings.TLSStrictTransportMaxAge = 6000
	})

	web := New(s, s.AppOptions, s.Router)
	if err != nil {
		panic(err)
	}
	handler := web.NewHandler(handlerForHTTPSecureTransport)

	request := httptest.NewRequest("GET", "/api/v4/test", nil)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	header := response.Header().Get("Strict-Transport-Security")

	if header == "" {
		t.Errorf("Strict-Transport-Security expected but not existent")
	}

	if header != "max-age=6000" {
		t.Errorf("Expected max-age=6000, got %s", header)
	}

	a.UpdateConfig(func(config *model.Config) {
		*config.ServiceSettings.TLSStrictTransport = false
	})

	request = httptest.NewRequest("GET", "/api/v4/test", nil)

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	header = response.Header().Get("Strict-Transport-Security")

	if header != "" {
		t.Errorf("Strict-Transport-Security header is not expected, but returned")
	}
}

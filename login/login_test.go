/*
 * Copyright (C) 2018-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under
 * the terms of the under the Apache License, Version 2.0 (the "License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package login_test

import (
	"github.com/pivotal-cf/service-instance-reaper/login"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-instance-reaper/login/loginfakes"
	"net/http"
	"errors"
	"io/ioutil"
	"bytes"
)

var _ = Describe("Login", func() {
	const (
		testApiUrl       = "example.com"
		testUser         = "username"
		testPassword     = "password"
		testErrorMessage = "There's many a slip 'twixt the cup and the lip"
	)

	var (
		fakeClient *loginfakes.FakeClient
		apiUrl     string
		token      string
		err        error
		testError  error
	)

	BeforeEach(func() {
		fakeClient = &loginfakes.FakeClient{}
		apiUrl = testApiUrl
		testError = errors.New(testErrorMessage)
	})

	JustBeforeEach(func() {
		token, err = login.GetOauthToken(fakeClient, apiUrl, testUser, testPassword)
	})

	Context("when the API URL is invalid", func() {
		BeforeEach(func() {
			apiUrl = ":"
		})

		It("should return a suitable error", func() {
			Expect(err.Error()).To(ContainSubstring("missing protocol scheme"))
		})
	})

	Context("when /v2/info fails with an error", func() {
		BeforeEach(func() {
			fakeClient.DoReturnsOnCall(0, &http.Response{}, testError)
		})

		It("should percolate the error", func() {
			Expect(err.Error()).To(ContainSubstring(testErrorMessage))
		})
	})

	Context("when /v2/info fails with an invalid HTTP response", func() {
		BeforeEach(func() {
			response := &http.Response{
				StatusCode: http.StatusBadGateway,
				Status:     "HTTP 502",
			}
			fakeClient.DoReturnsOnCall(0, response, nil)
		})

		It("should percolate the error", func() {
			Expect(err.Error()).To(ContainSubstring("request failed: HTTP 502"))
		})
	})

	Context("when /v2/info returns a body that cannot be read", func() {
		BeforeEach(func() {
			response := &http.Response{
				StatusCode: http.StatusOK,
				Body:       badReader{},
			}
			fakeClient.DoReturnsOnCall(0, response, nil)
		})

		It("should percolate the error", func() {
			Expect(err).To(MatchError("/v2/info failure: read error"))
		})
	})

	Context("when /v2/info returns invalid JSON", func() {
		BeforeEach(func() {
			response := &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString("")),
			}
			fakeClient.DoReturnsOnCall(0, response, nil)
		})

		It("should percolate the error", func() {
			Expect(err).To(MatchError("/v2/info failure: EOF"))
		})
	})

	Context("when /v2/info returns valid JSON", func() {
		BeforeEach(func() {
			response := &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"authorization_endpoint":"auth.endpoint"}`)),
			}
			fakeClient.DoReturnsOnCall(0, response, nil)
			fakeClient.DoReturnsOnCall(1, &http.Response{}, testError)
		})

		It("should GET /v2/info", func() { // Test happy path in this context to avoid panic
			request := fakeClient.DoArgsForCall(0)
			Expect(request.Method).To(Equal("GET"))
			Expect(request.URL.String()).To(Equal("example.com/v2/info"))
			Expect(request.Header.Get("Accept")).To(Equal("application/json"))
		})

		Context("when the authorisation endpoint is invalid", func() {
			BeforeEach(func() {
				response := &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewBufferString(`{"authorization_endpoint":":"}`)),
				}
				fakeClient.DoReturnsOnCall(0, response, nil)
			})

			It("should return a suitable error", func() {
				Expect(err.Error()).To(ContainSubstring("missing protocol scheme"))
			})
		})

		Context("when /login fails with an error", func() {
			BeforeEach(func() {
				fakeClient.DoReturnsOnCall(1, &http.Response{}, testError)
			})

			It("should percolate the error", func() {
				Expect(err.Error()).To(ContainSubstring(testErrorMessage))
			})
		})

		Context("when /login returns valid JSON", func() {
			BeforeEach(func() {
				response := &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewBufferString(`{"links": {"login": "login.endpoint"}}`)),
				}
				fakeClient.DoReturnsOnCall(1, response, nil)
				fakeClient.DoReturnsOnCall(2, &http.Response{}, testError)
			})

			It("should GET /login", func() { // Test happy path in this context to avoid panic
				request := fakeClient.DoArgsForCall(1)
				Expect(request.Method).To(Equal("GET"))
				Expect(request.URL.String()).To(Equal("auth.endpoint/login"))
				Expect(request.Header.Get("Accept")).To(Equal("application/json"))
			})

			Context("when the login endpoint is invalid", func() {
				BeforeEach(func() {
					response := &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewBufferString(`{"links": {"login": ":"}}`)),
					}
					fakeClient.DoReturnsOnCall(1, response, nil)
				})

				It("should return a suitable error", func() {
					Expect(err.Error()).To(ContainSubstring("missing protocol scheme"))
				})
			})

			Context("when /login returns valid JSON", func() {
				BeforeEach(func() {
					response := &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewBufferString(`{"access_token":"some-token"}`)),
					}
					fakeClient.DoReturnsOnCall(2, response, nil)
				})

				It("should POST /oauth/token", func() { // Test happy path in this context to avoid panic
					request := fakeClient.DoArgsForCall(2)
					Expect(request.Method).To(Equal("POST"))
					Expect(request.URL.String()).To(Equal("login.endpoint/oauth/token"))
					Expect(request.Header.Get("Content-Type")).To(Equal("application/x-www-form-urlencoded"))
					Expect(request.Header.Get("Accept")).To(Equal("application/json"))
					Expect(request.Header.Get("Authorization")).To(Equal("Basic Y2Y6"))

					body, bodyErr := ioutil.ReadAll(request.Body)
					Expect(bodyErr).NotTo(HaveOccurred())
					Expect(string(body)).To(Equal("grant_type=password&password=password&scope=&username=username"))
				})

				It("should return the access token", func() {
				    Expect(err).NotTo(HaveOccurred())
					Expect(token).To(Equal("some-token"))
				})
			})
		})
	})
})

type badReader struct{}

func (badReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func (badReader) Close() error {
	return nil
}

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
package cloudfoundry_test

import (
	"bytes"
	"errors"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-instance-reaper/cloudfoundry"
	"github.com/pivotal-cf/service-instance-reaper/httpclient/httpclientfakes"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	testAccessToken         = "xxx"
	testServiceName         = "config-server"
	testServiceGuid         = "test-service-guid"
	testServicePlanGuid     = "test-service-plan-guid"
	testServiceInstanceGuid = "test-service-instance-guid"
	testApiUrl              = "example.com"
)

var (
	authClient *httpclientfakes.FakeAuthenticatedClient
	cf         cloudfoundry.Client
	testError  = errors.New("test error")
)

var _ = Describe("CF", func() {

	var (
		testError  = errors.New("test error")
		fakeClient *httpclientfakes.FakeHttpClient
	)

	Describe("unauthentictated functions", func() {
		Describe("GetOauthToken", func() {
			const (
				testUser     = "username"
				testPassword = "password"
			)

			var (
				token  string
				err    error
				apiUrl string
			)

			BeforeEach(func() {
				fakeClient = &httpclientfakes.FakeHttpClient{}
				apiUrl = testApiUrl
			})

			JustBeforeEach(func() {
				token, err = cloudfoundry.GetOauthToken(fakeClient, apiUrl, testUser, testPassword)
			})

			Context("when the API URL is invalid", func() {
				BeforeEach(func() {
					apiUrl = ":"
				})

				It("returns a suitable error", func() {
					Expect(err).To(MatchError(ContainSubstring("missing protocol scheme")))
				})
			})

			Context("when /v2/info fails with an error", func() {
				BeforeEach(func() {
					fakeClient.DoReturnsOnCall(0, &http.Response{}, testError)
				})

				It("percolates the error", func() {
					Expect(err).To(MatchError(fmt.Sprintf("/v2/info failure: %s", testError)))
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

				It("percolates the error", func() {
					Expect(err).To(MatchError(ContainSubstring("request failed: HTTP 502")))
				})
			})

			Context("when /v2/info returns a body that cannot be read", func() {
				BeforeEach(func() {
					response := &http.Response{
						StatusCode: http.StatusOK,
						Body:       brokenReadCloser{},
					}
					fakeClient.DoReturnsOnCall(0, response, nil)
				})

				It("percolates the error", func() {
					Expect(err).To(MatchError("/v2/info failure: read failed"))
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

				It("percolates the error", func() {
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

				It("GETs /v2/info", func() { // Test happy path in this context to avoid panic
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

					It("returns a suitable error", func() {
						Expect(err).To(MatchError(ContainSubstring("missing protocol scheme")))
					})
				})

				Context("when /login fails with an error", func() {
					BeforeEach(func() {
						fakeClient.DoReturnsOnCall(1, &http.Response{}, testError)
					})

					It("percolates the error", func() {
						Expect(err).To(MatchError(fmt.Sprintf("/login failure: %s", testError)))
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

					It("GETs /login", func() { // Test happy path in this context to avoid panic
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

						It("returns a suitable error", func() {
							Expect(err).To(MatchError(ContainSubstring("missing protocol scheme")))
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

						It("POSTs /oauth/token", func() { // Test happy path in this context to avoid panic
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

						It("returns the access token", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(token).To(Equal("some-token"))
						})
					})
				})
			})
		})
	})

	Describe("authenticated functions", func() {
		BeforeEach(func() {
			authClient = &httpclientfakes.FakeAuthenticatedClient{}
			cf = cloudfoundry.NewClient(authClient, testApiUrl, testAccessToken)
		})

		Describe("GetServices", func() {
			assertStandardHttpGetErrorHandling(
				func() (interface{}, error) { return cf.GetServices(testServiceName) },
				fmt.Sprintf("/v2/services?q=label:%s", testServiceName),
			)

			Context("when the CF API call is successful", func() {
				BeforeEach(func() {
					servicesJson := `{
	"resources": [
		{
			"metadata": {
				"guid": "service-guid-0",
				"created_at": "service-created-at-0"
			}
		},
		{
			"metadata": {
				"guid": "service-guid-1",
				"created_at": "service-created-at-1"
			}
		}
	]
}`
					authClient.DoAuthenticatedGetReturns(stringReadCloser(servicesJson), http.StatusOK, nil)
				})

				It("returns a list of services with the given name", func() {
					services, err := cf.GetServices(testServiceName)
					Expect(err).NotTo(HaveOccurred())

					Expect(authClient.DoAuthenticatedGetCallCount()).To(Equal(1), "Incorrect number of calls to CF API")

					url, accessToken := authClient.DoAuthenticatedGetArgsForCall(0)
					Expect(url).To(Equal(fmt.Sprintf("%s/v2/services?q=label:%s", testApiUrl, testServiceName)))
					Expect(accessToken).To(Equal(testAccessToken))

					Expect(services).NotTo(BeNil())
					Expect(len(services)).To(Equal(2), "Unexpected number of services returned")
					Expect(services[0].Metadata.Guid).To(Equal("service-guid-0"))
					Expect(services[0].Metadata.CreatedAt).To(Equal("service-created-at-0"))
					Expect(services[1].Metadata.Guid).To(Equal("service-guid-1"))
					Expect(services[1].Metadata.CreatedAt).To(Equal("service-created-at-1"))
				})
			})
		})

		Describe("GetServicePlans", func() {
			assertStandardHttpGetErrorHandling(
				func() (interface{}, error) { return cf.GetServicePlans(testServiceGuid) },
				fmt.Sprintf("/v2/services/%s/service_plans?results-per-page=%d", testServiceGuid, cloudfoundry.MaximumResultsPerPage),
			)

			Context("when the CF API call is successful", func() {
				BeforeEach(func() {
					servicePlansJson := []string{
						fmt.Sprintf(`{
  "next_url": "/v2/services/%s/service_plans?page=2&results-per-page=%d",
  "resources": [
    {
      "metadata": {
        "guid": "service-plan-guid-0",
        "created_at": "service-plan-created-at-0"
      },
      "entity": {
        "name": "service-plan-name-0",
        "free": true
      }
    }
  ]
}`, testServiceGuid, cloudfoundry.MaximumResultsPerPage),
						`{
  "resources": [
    {
      "metadata": {
        "guid": "service-plan-guid-1",
        "created_at": "service-plan-created-at-1"
      },
      "entity": {
        "name": "service-plan-name-1",
        "free": false
      }
    }
  ]
}`,
					}
					authClient.DoAuthenticatedGetReturnsOnCall(0, stringReadCloser(servicePlansJson[0]), http.StatusOK, nil)
					authClient.DoAuthenticatedGetReturnsOnCall(1, stringReadCloser(servicePlansJson[1]), http.StatusOK, nil)
				})

				It("returns a list of plans for the service with the given guid", func() {
					servicePlans, err := cf.GetServicePlans(testServiceGuid)
					Expect(err).NotTo(HaveOccurred())

					Expect(authClient.DoAuthenticatedGetCallCount()).To(Equal(2), "Incorrect number of calls to CF API")
					url, accessToken := authClient.DoAuthenticatedGetArgsForCall(0)
					Expect(url).To(Equal(fmt.Sprintf("%s/v2/services/%s/service_plans?results-per-page=%d", testApiUrl, testServiceGuid, cloudfoundry.MaximumResultsPerPage)))
					Expect(accessToken).To(Equal(testAccessToken))

					url, accessToken = authClient.DoAuthenticatedGetArgsForCall(1)
					Expect(url).To(Equal(fmt.Sprintf("%s/v2/services/%s/service_plans?page=2&results-per-page=%d", testApiUrl, testServiceGuid, cloudfoundry.MaximumResultsPerPage)))
					Expect(accessToken).To(Equal(testAccessToken))

					Expect(servicePlans).NotTo(BeNil())
					Expect(len(servicePlans)).To(Equal(2), "Unexpected number of service plans returned")
					Expect(servicePlans[0].Metadata.Guid).To(Equal("service-plan-guid-0"))
					Expect(servicePlans[0].Metadata.CreatedAt).To(Equal("service-plan-created-at-0"))
					Expect(servicePlans[0].Entity.Name).To(Equal("service-plan-name-0"))
					Expect(servicePlans[0].Entity.Free).To(BeTrue())
					Expect(servicePlans[1].Metadata.Guid).To(Equal("service-plan-guid-1"))
					Expect(servicePlans[1].Metadata.CreatedAt).To(Equal("service-plan-created-at-1"))
					Expect(servicePlans[1].Entity.Name).To(Equal("service-plan-name-1"))
					Expect(servicePlans[1].Entity.Free).To(BeFalse())
				})
			})
		})

		Describe("GetServicePlanInstances", func() {
			assertPaginatedHttpGetErrorHandling(
				func() (interface{}, chan error) {
					return cf.GetServicePlanInstances(testServicePlanGuid)
				},
				fmt.Sprintf("/v2/service_plans/%s/service_instances?results-per-page=%d", testServicePlanGuid, cloudfoundry.MaximumResultsPerPage),
			)

			Context("when the CF API call is successful", func() {
				BeforeEach(func() {
					servicePlanInstancesJson := `{
  "resources": [
    {
      "metadata": {
        "guid": "service-plan-instance-guid-0",
        "created_at": "service-plan-instance-created-at-0"
      },
      "entity": {
        "name": "service-plan-instance-name-0"
      }
    },
    {
      "metadata": {
        "guid": "service-plan-instance-guid-1",
        "created_at": "service-plan-instance-created-at-1"
      },
      "entity": {
        "name": "service-plan-instance-name-1"
      }
    }
  ]
}`
					authClient.DoAuthenticatedGetReturns(stringReadCloser(servicePlanInstancesJson), http.StatusOK, nil)
				})

				It("returns a list of plans for the service with the given guid", func() {
					servicePlanInstances, errors := cf.GetServicePlanInstances(testServicePlanGuid)

					var serviceInstance cloudfoundry.ServiceInstance
					Eventually(servicePlanInstances).Should(Receive(&serviceInstance))
					Expect(serviceInstance.Metadata.Guid).To(Equal("service-plan-instance-guid-0"))
					Expect(serviceInstance.Metadata.CreatedAt).To(Equal("service-plan-instance-created-at-0"))
					Expect(serviceInstance.Entity.Name).To(Equal("service-plan-instance-name-0"))

					Eventually(servicePlanInstances).Should(Receive(&serviceInstance))
					Expect(serviceInstance.Metadata.Guid).To(Equal("service-plan-instance-guid-1"))
					Expect(serviceInstance.Metadata.CreatedAt).To(Equal("service-plan-instance-created-at-1"))
					Expect(serviceInstance.Entity.Name).To(Equal("service-plan-instance-name-1"))

					Eventually(servicePlanInstances).Should(BeClosed())

					Expect(authClient.DoAuthenticatedGetCallCount()).To(Equal(1), "Incorrect number of calls to CF API")

					url, accessToken := authClient.DoAuthenticatedGetArgsForCall(0)
					Expect(url).To(Equal(fmt.Sprintf("%s/v2/service_plans/%s/service_instances?results-per-page=%d", testApiUrl, testServicePlanGuid, cloudfoundry.MaximumResultsPerPage)))
					Expect(accessToken).To(Equal(testAccessToken))

					Eventually(errors).Should(BeClosed())
					Expect(len(errors)).To(BeZero(), "No errors should have occurred")
				})
			})
		})

		Describe("DeleteServiceInstance", func() {
			Context("when the recursive flag is false", func() {
				assertStandardHttpDeleteErrorHandling(
					func() error { return cf.DeleteServiceInstance(testServiceInstanceGuid, false) },
					fmt.Sprintf("/v2/service_instances/%s?accepts_incomplete=true;async=true;recursive=false", testServiceInstanceGuid),
				)

				Context("when the API call is successful", func() {
					BeforeEach(func() {
						authClient.DoAuthenticatedDeleteReturns(http.StatusNoContent, nil)
					})

					It("succeeds", func() {
						Expect(cf.DeleteServiceInstance(testServiceInstanceGuid, false)).To(Succeed())
						Expect(authClient.DoAuthenticatedDeleteCallCount()).To(Equal(1), "Unexpected number of delete API calls")
						url, accessToken := authClient.DoAuthenticatedDeleteArgsForCall(0)
						Expect(url).To(Equal(fmt.Sprintf("%s/v2/service_instances/%s?accepts_incomplete=true;async=true;recursive=false", testApiUrl, testServiceInstanceGuid)))
						Expect(accessToken).To(Equal(testAccessToken))
					})
				})
			})

			Context("when the recursive flag is true", func() {
				assertStandardHttpDeleteErrorHandling(
					func() error { return cf.DeleteServiceInstance(testServiceInstanceGuid, true) },
					fmt.Sprintf("/v2/service_instances/%s?accepts_incomplete=true;async=true;recursive=true", testServiceInstanceGuid),
				)

				Context("when the API call is successful", func() {
					BeforeEach(func() {
						authClient.DoAuthenticatedDeleteReturns(http.StatusNoContent, nil)
					})

					It("succeeds", func() {
						Expect(cf.DeleteServiceInstance(testServiceInstanceGuid, true)).To(Succeed())
						Expect(authClient.DoAuthenticatedDeleteCallCount()).To(Equal(1), "Unexpected number of delete API calls")
						url, accessToken := authClient.DoAuthenticatedDeleteArgsForCall(0)
						Expect(url).To(Equal(fmt.Sprintf("%s/v2/service_instances/%s?accepts_incomplete=true;async=true;recursive=true", testApiUrl, testServiceInstanceGuid)))
						Expect(accessToken).To(Equal(testAccessToken))
					})
				})
			})
		})
	})
})

func assertStandardHttpDeleteErrorHandling(cfDeleteOperation func() error, expectedEndpoint string) {
	Context("when call to the CF API fails", func() {
		BeforeEach(func() {
			authClient.DoAuthenticatedDeleteReturns(0, testError)
		})

		It("returns the error", func() {
			Expect(cfDeleteOperation()).To(MatchError(fmt.Sprintf("DELETE %s failed: test error", expectedEndpoint)))
		})
	})

	Context("when a non-OK HTTP status code is returned from the CF API", func() {
		BeforeEach(func() {
			authClient.DoAuthenticatedDeleteReturns(http.StatusInternalServerError, nil)
		})

		It("returns the error", func() {
			Expect(cfDeleteOperation()).To(MatchError(fmt.Sprintf("DELETE %s failed: HTTP status 500", expectedEndpoint)))
		})
	})
}

func assertStandardHttpGetErrorHandling(cfGetOperation func() (interface{}, error), expectedEndpoint string) {
	Describe("error handling", func() {
		var err error

		JustBeforeEach(func() {
			_, err = cfGetOperation()
		})

		Context("when call to the CF API fails", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(nil, 0, testError)
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(fmt.Sprintf("GET %s failed: test error", expectedEndpoint)))
			})
		})

		Context("when a non-OK HTTP status code is returned from the CF API", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(nil, http.StatusInternalServerError, nil)
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(fmt.Sprintf("GET %s failed: HTTP status 500", expectedEndpoint)))
			})
		})

		Context("when the CF API call returns an empty body", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(nil, http.StatusOK, nil)
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(fmt.Sprintf("GET %s response body missing", expectedEndpoint)))
			})
		})

		Context("when the response body cannot be read", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(brokenReadCloser{}, http.StatusOK, nil)
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(fmt.Sprintf("cannot read GET %s response body: read failed", expectedEndpoint)))
			})
		})

		Context("when the CF API call returns an invalid body", func() {
			BeforeEach(func() {
				authClient.DoAuthenticatedGetReturns(stringReadCloser("rubbish"), http.StatusOK, nil)
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(fmt.Sprintf("invalid GET %s response JSON: invalid character 'r' looking for beginning of value", expectedEndpoint)))
			})
		})
	})
}

func assertPaginatedHttpGetErrorHandling(cfGetOperation func() (interface{}, chan error), expectedEndpoint string) {
	assertStandardHttpGetErrorHandling(func() (interface{}, error) {
		_, errorChannel := cfGetOperation()

		var err error
		Eventually(errorChannel).Should(Receive(&err))
		Eventually(errorChannel).Should(BeClosed())

		return nil, err
	}, expectedEndpoint)
}

func stringReadCloser(data string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBufferString(data))
}

type brokenReadCloser struct{}

func (brc brokenReadCloser) Read(p []byte) (n int, err error) {
	return 0, errors.New("read failed")
}

func (brc brokenReadCloser) Close() error { return nil }

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
package integration_test

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path"
	"strings"
	"time"
)

const (
	username    = "username"
	password    = "password"
	serviceName = "service-name"
	planName    = "service-plan-name-0"
	age         = "10"
	accessToken = "access-token"
)

var (
	session         *Session
	args            []string
	fakeCfApiServer *httptest.Server
	httpHandler     http.HandlerFunc
)

// Tests to assert end-to-end wiring, including the main function
// For exhaustive behavioural tests, check out the units
var _ = Describe("Integration tests", func() {

	var (
		deletedServices = make([]string, 0)
	)

	BeforeEach(func() {
		httpHandler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/info":
				handleGet(rw, r, getV2Info)
			case "/uaa/login":
				handleGet(rw, r, getUaaLogin)
			case "/uaa/oauth/token":
				handlePost(rw, r, postUaaOAuthToken)
			case "/v2/services":
				handleGet(rw, r, getServices)
			case "/v2/services/service-guid-0/service_plans":
				handleGet(rw, r, getServicePlans)
			case "/v2/service_plans/service-plan-guid-0/service_instances":
				handleGet(rw, r, getServiceInstances)
			default:
				if r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v2/service_instances") {
					deletedServices = handleDeleteServiceInstance(deletedServices, rw, r)
				}
			}
		})

		fakeCfApiServer = httptest.NewTLSServer(httpHandler)

		args = []string{
			"-u", username,
			"-p", password,
			"-skip-ssl-validation",
			"-reap",
			fakeCfApiServer.URL,
			serviceName,
			planName,
			age,
		}
	})

	JustBeforeEach(func() {
		command := exec.Command(pathToReaper, args...)
		var err error
		session, err = Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		fakeCfApiServer.Close()
	})

	Describe("Reaper", func() {
		It("successfully reaps some services", func() {
			Eventually(session, 1*time.Second).Should(Exit(0))
			Expect(deletedServices).To(ConsistOf("service-plan-instance-guid-0", "service-plan-instance-guid-1"))
		})
	})
})

func handleGet(rw http.ResponseWriter, r *http.Request, handlerFunc http.HandlerFunc) {
	handle(http.MethodGet, rw, r, handlerFunc)
}

func handlePost(rw http.ResponseWriter, r *http.Request, handlerFunc http.HandlerFunc) {
	handle(http.MethodPost, rw, r, handlerFunc)
}

func handle(method string, rw http.ResponseWriter, r *http.Request, handlerFunc http.HandlerFunc) {
	if r.Method == method {
		handlerFunc(rw, r)
	}
}

func getV2Info(rw http.ResponseWriter, r *http.Request) {
	jsonBytes, err := json.Marshal(map[string]string{
		"authorization_endpoint": fmt.Sprintf("https://%s/uaa", r.Host),
	})
	if err != nil {
		panic("test data json marshalling failed")
	}
	rw.Write(jsonBytes)
}

func getUaaLogin(rw http.ResponseWriter, r *http.Request) {
	jsonBytes, err := json.Marshal(map[string]interface{}{
		"links": map[string]string{
			"login": fmt.Sprintf("https://%s/uaa", r.Host),
		},
	})
	if err != nil {
		panic("test data json marshalling failed")
	}
	rw.Write(jsonBytes)
}

func postUaaOAuthToken(rw http.ResponseWriter, _ *http.Request) {
	jsonBytes, err := json.Marshal(map[string]string{
		"access_token": accessToken,
	})
	if err != nil {
		panic("test data json marshalling failed")
	}
	rw.Write(jsonBytes)
}

func getServices(rw http.ResponseWriter, _ *http.Request) {
	jsonBytes := []byte(`{
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
}`)
	rw.Write(jsonBytes)
}

func getServicePlans(rw http.ResponseWriter, _ *http.Request) {
	jsonBytes := []byte(`{
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
    },
    {
      "metadata": {
        "guid": "service-plan-guid-1",
        "created_at": "service-plan-created-at-0"
      },
      "entity": {
        "name": "service-plan-name-1",
        "free": true
      }
    }
  ]
}`)
	rw.Write(jsonBytes)
}

func getServiceInstances(rw http.ResponseWriter, _ *http.Request) {
	jsonBytes := []byte(`{
  "resources": [
    {
      "metadata": {
        "guid": "service-plan-instance-guid-0",
        "created_at": "2015-01-01T10:00:00Z"
      },
      "entity": {
        "name": "service-plan-instance-name-0"
      }
    },
    {
      "metadata": {
        "guid": "service-plan-instance-guid-1",
        "created_at": "2015-01-01T11:00:00Z"
      },
      "entity": {
        "name": "service-plan-instance-name-1"
      }
    }
  ]
}`)
	rw.Write(jsonBytes)
}

func handleDeleteServiceInstance(deletedServices []string, rw http.ResponseWriter, r *http.Request) []string {
	rw.WriteHeader(http.StatusNoContent)
	return append(deletedServices, path.Base(r.URL.Path))
}

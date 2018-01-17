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
package reaper

import (
	"fmt"
	"io/ioutil"
	"time"
	"github.com/pivotal-cf/service-instance-reaper/httpclient"
	"net/http"
	"encoding/json"
)

// FIXME: support multi-page responses
const MAXIMUM_RESULTS_PER_PAGE = 100

func Reap(authClient httpclient.AuthenticatedClient, apiUrl string, serviceName string, accessToken string, expiryInterval time.Duration, reap bool, fatal func(string)) {
	bodyReader, statusCode, err := authClient.DoAuthenticatedGet(apiUrl+fmt.Sprintf("/v2/services?q=label:%s", serviceName), accessToken)
	if err != nil {
		fatalError("GET /v2/services failed", err, fatal)
	}
	if statusCode != http.StatusOK {
		fatal(fmt.Sprintf("GET /v2/services failed: %d", statusCode))
	}

	if bodyReader == nil {
		fatal("GET /v2/services response body missing")
	}
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		fatalError("Cannot read GET /v2/services response body: %s", err, fatal)
	}

	var listServicesResp ListServicesResp
	err = json.Unmarshal(body, &listServicesResp)
	if err != nil {
		fatalError("Invalid GET /v2/services response JSON", err, fatal)
	}

	serviceGuid := listServicesResp.Resources[0].Metadata.Guid

	bodyReader, statusCode, err = authClient.DoAuthenticatedGet(apiUrl+fmt.Sprintf("/v2/services/%s/service_plans?results-per-page=%d", serviceGuid, MAXIMUM_RESULTS_PER_PAGE), accessToken)
	if err != nil {
		fatalError("GET /v2/services/[service GUID]/service_plans failed", err, fatal)
	}
	if statusCode != http.StatusOK {
		fatal(fmt.Sprintf("GET /v2/services/[service GUID]/service_plans failed: %d", statusCode))
	}

	if bodyReader == nil {
		fatal("GET /v2/services/[service GUID]/service_plans response body missing")
	}
	body, err = ioutil.ReadAll(bodyReader)
	if err != nil {
		fatalError("Cannot read GET /v2/services/[service GUID]/service_plans response body: %s", err, fatal)
	}

	var listServicePlansResp ListServicePlansResp
	err = json.Unmarshal(body, &listServicePlansResp)
	if err != nil {
		fatalError("Invalid GET /v2/services/[service GUID]/service_plans response JSON", err, fatal)
	}

	for _, sp := range listServicePlansResp.Resources {
		if sp.Entity.Free {
			bodyReader, statusCode, err = authClient.DoAuthenticatedGet(apiUrl+fmt.Sprintf("/v2/service_plans/%s/service_instances?results-per-page=%d", sp.Metadata.Guid, MAXIMUM_RESULTS_PER_PAGE), accessToken)
			if err != nil {
				fatalError("GET /v2/service_plans/[service plan GUID]/service_instances failed", err, fatal)
			}
			if statusCode != http.StatusOK {
				fatal(fmt.Sprintf("GET /v2/service_plans/[service plan GUID]/service_instances: %d", statusCode))
			}

			if bodyReader == nil {
				fatal("GET /v2/service_plans/[service plan GUID]/service_instances response body missing")
			}
			body, err = ioutil.ReadAll(bodyReader)
			if err != nil {
				fatalError("Cannot read GET /v2/service_plans/[service plan GUID]/service_instances response body: %s", err, fatal)
			}

			var servicePlanListServiceInstancesResp ServicePlanListServiceInstancesResp
			err = json.Unmarshal(body, &servicePlanListServiceInstancesResp)
			if err != nil {
				fatalError("Invalid GET /v2/service_plans/[service plan GUID]/service_instances response JSON", err, fatal)
			}

			for _, si := range servicePlanListServiceInstancesResp.Resources {
				if expired(si.Metadata.CreatedAt, expiryInterval, fatal) {
					if reap {
						// TODO: delete the service instance and add force option to do recursive delete
					} else {
						fmt.Printf("%s %s", si.Entity.Name, si.Metadata.Guid)
					}
				}
			}

		}
	}
}

func fatalError(message string, err error, fatal func(string)) {
	fatal(fmt.Sprintf("%s: %s", message, err))
}

func expired(creationTimeString string, expiryInterval time.Duration, fatal func(string)) bool {
	creationTime, err := time.Parse(time.RFC3339, creationTimeString)
	if err != nil {
		fatalError("Invalid service instance creation time", err, fatal)
	}
	expiryTime := creationTime.Add(expiryInterval)
	return time.Now().After(expiryTime)
}

type Metadata struct {
	Guid      string
	CreatedAt string `json:"created_at"`
}

type ListServicesResp struct {
	Resources []struct {
		Metadata Metadata
	}
}

type ListServicePlansResp struct {
	Resources []struct {
		Metadata Metadata
		Entity struct {
			Name string
			Free bool
		}
	}
}

type ServicePlanListServiceInstancesResp struct {
	Resources []struct {
		Metadata Metadata
		Entity struct {
			Name string
		}
	}
}

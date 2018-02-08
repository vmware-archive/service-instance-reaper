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
package cloudfoundry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pivotal-cf/service-instance-reaper/httpclient"
	"io/ioutil"
	"net/http"
	"strings"
)

const MaximumResultsPerPage = 50

//go:generate counterfeiter . Client
type Client interface {
	GetServices(serviceName string) ([]Service, error)
	GetServicePlans(serviceGuid string) ([]ServicePlan, error)
	GetServicePlanInstances(servicePlanGuid string) (chan ServiceInstance, chan error)
	DeleteServiceInstance(serviceInstanceGuid string, recursive bool) error
}

type client struct {
	authClient  httpclient.AuthenticatedClient
	apiUrl      string
	accessToken string
}

func GetOauthToken(client httpclient.HttpClient, apiUrl string, username string, password string) (string, error) {
	request, err := http.NewRequest("GET", apiUrl+"/v2/info", nil)
	if err != nil {
		return "", fmt.Errorf("unable to build http request: %s", err)
	}
	request.Header.Add("Accept", "application/json")
	var infoResp infoResponse

	err = do(client, request, &infoResp)
	if err != nil {
		return "", fmt.Errorf("/v2/info failure: %s", err)
	}

	request, err = http.NewRequest("GET", infoResp.AuthorisationEndpoint+"/login", nil)
	if err != nil {
		return "", err
	}
	request.Header.Add("Accept", "application/json")
	var loginResp loginResponse
	err = do(client, request, &loginResp)
	if err != nil {
		return "", fmt.Errorf("/login failure: %s", err)
	}

	body := fmt.Sprintf("grant_type=password&password=%s&scope=&username=%s", password, username)
	request, err = http.NewRequest("POST", loginResp.Links.Login+"/oauth/token", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Accept", "application/json")
	accessToken := "Basic " + base64.StdEncoding.EncodeToString([]byte("cf:"))
	request.Header.Add("Authorization", accessToken)
	var tokenResp tokenResponse
	err = do(client, request, &tokenResp)
	if err != nil {
		return "", fmt.Errorf("/outh/token failure: %s", err)
	}

	return tokenResp.AccessToken, nil
}

func NewClient(authClient httpclient.AuthenticatedClient, apiUrl, accessToken string) Client {
	return &client{
		authClient:  authClient,
		apiUrl:      apiUrl,
		accessToken: accessToken,
	}
}

func (cf *client) GetServices(serviceName string) (services []Service, err error) {
	var servicesResponse listServicesResponse
	err = cf.get(fmt.Sprintf("/v2/services?q=label:%s", serviceName), &servicesResponse)
	services = servicesResponse.Resources
	return
}

func (cf *client) GetServicePlans(serviceGuid string) (servicePlans []ServicePlan, err error) {
	servicePlans = make([]ServicePlan, 0)
	endpoint := fmt.Sprintf("/v2/services/%s/service_plans?results-per-page=%d", serviceGuid, MaximumResultsPerPage)

	for endpoint != "" {
		var servicePlanResponse listServicePlansResponse
		err = cf.get(endpoint, &servicePlanResponse)
		if err != nil {
			return
		}

		for _, servicePlan := range servicePlanResponse.Resources {
			servicePlans = append(servicePlans, servicePlan)
		}

		endpoint = servicePlanResponse.NextUrl
	}

	return
}

func (cf *client) GetServicePlanInstances(servicePlanGuid string) (servicePlanInstances chan ServiceInstance, errorChannel chan error) {
	servicePlanInstances = make(chan ServiceInstance, MaximumResultsPerPage)
	errorChannel = make(chan error, 1)

	endpoint := fmt.Sprintf("/v2/service_plans/%s/service_instances?results-per-page=%d", servicePlanGuid, MaximumResultsPerPage)

	go func() {
		defer close(servicePlanInstances)
		defer close(errorChannel)

		for endpoint != "" {
			var servicePlanInstancesResponse listServicePlanInstancesResponse
			err := cf.get(endpoint, &servicePlanInstancesResponse)
			if err != nil {
				errorChannel <- err
				return
			}

			for _, servicePlanInstance := range servicePlanInstancesResponse.Resources {
				servicePlanInstances <- servicePlanInstance
			}

			endpoint = servicePlanInstancesResponse.NextUrl
		}
	}()

	return
}

func (cf *client) DeleteServiceInstance(serviceInstanceGuid string, recursive bool) (err error) {
	return cf.delete(fmt.Sprintf("/v2/service_instances/%s?accepts_incomplete=true;async=true;recursive=%t", serviceInstanceGuid, recursive))
}

func (cf *client) get(endpoint string, response interface{}) error {
	bodyReader, statusCode, err := cf.authClient.DoAuthenticatedGet(cf.apiUrl+endpoint, cf.accessToken)

	if err != nil {
		return fmt.Errorf("GET %s failed: %s", endpoint, err)
	}

	if statusCode != http.StatusOK {
		return fmt.Errorf("GET %s failed: HTTP status %d", endpoint, statusCode)
	}

	if bodyReader == nil {
		return fmt.Errorf("GET %s response body missing", endpoint)
	}

	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return fmt.Errorf("cannot read GET %s response body: %s", endpoint, err)
	}

	err = json.Unmarshal(body, response)
	if err != nil {
		return fmt.Errorf("invalid GET %s response JSON: %s", endpoint, err)
	}

	return nil
}

func (cf *client) delete(endpoint string) error {
	statusCode, err := cf.authClient.DoAuthenticatedDelete(cf.apiUrl+endpoint, cf.accessToken)
	if err != nil {
		return fmt.Errorf("DELETE %s failed: %s", endpoint, err)
	}

	if statusCode != http.StatusNoContent {
		return fmt.Errorf("DELETE %s failed: HTTP status %d", endpoint, statusCode)
	}

	return nil
}

func do(client httpclient.HttpClient, request *http.Request, v interface{}) error {
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", response.Status)
	}
	defer response.Body.Close()

	return json.NewDecoder(response.Body).Decode(v)
}

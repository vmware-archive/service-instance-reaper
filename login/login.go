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
package login

import (
	"fmt"
	"strings"
	"encoding/base64"
	"net/http"
	"encoding/json"
)

//go:generate counterfeiter -o loginfakes/fake_client.go . Client
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type InfoResp struct {
	AuthorisationEndpoint string `json:"authorization_endpoint"`
}

type LoginResp struct {
	Links struct {
		Login string
	}
}

type TokenResp struct {
	AccessToken string `json:"access_token"`
}

func GetOauthToken(client Client, apiUrl string, username string, password string) (string, error) {
	request, err := http.NewRequest("GET", apiUrl+"/v2/info", nil)
	if err != nil {
		return "", err
	}
	request.Header.Add("Accept", "application/json")
	var infoResp InfoResp
	err = do(client, request, &infoResp)
	if err != nil {
		return "", fmt.Errorf("/v2/info failure: %s", err)
	}

	request, err = http.NewRequest("GET", infoResp.AuthorisationEndpoint+"/login", nil)
	if err != nil {
		return "", err
	}
	request.Header.Add("Accept", "application/json")
	var loginResp LoginResp
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
	var tokenResp TokenResp
	err = do(client, request, &tokenResp)
	if err != nil {
		return "", fmt.Errorf("/outh/token failure: %s", err)
	}

	return tokenResp.AccessToken, nil
}

func do(client Client, request *http.Request, v interface{}) error {
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

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
package main

import (
	"fmt"
	"os"
	"net/http"
	"crypto/tls"
	"github.com/pivotal-cf/service-instance-reaper/login"
	"github.com/pivotal-cf/service-instance-reaper/httpclient"
	"github.com/pivotal-cf/service-instance-reaper/reaper"
	"github.com/pivotal-cf/service-instance-reaper/arg"
	"github.com/hako/durafmt"
)

func main() {
	username, password, skipSslValidation, reap, apiUrl, serviceName, expiryInterval := arg.Parse(os.Args, func(code int){os.Exit(code)})

	if !reap {
		fmt.Printf("DRY RUN ONLY!\n")
	}
	fmt.Printf("Reaping instances of service %s older than %s in %s as %s...\n", serviceName, durafmt.Parse(expiryInterval), apiUrl, username)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSslValidation},
	}
	client := &http.Client{Transport: tr}

	accessToken, err := login.GetOauthToken(client, apiUrl, username, password)
	if err != nil {
		fatalError("Authentication failed", err)
	}

	authClient := httpclient.NewAuthenticatedClient(client)
	reaper.Reap(authClient, apiUrl, serviceName, accessToken, expiryInterval, reap, fatal)
}

func fatalError(message string, err error) {
	fatal(fmt.Sprintf("%s: %s", message, err))
}

func fatal(message string) {
	fmt.Println(message)
	os.Exit(1)
}

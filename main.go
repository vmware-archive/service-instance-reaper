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
	"crypto/tls"
	"fmt"
	"github.com/hako/durafmt"
	"github.com/pivotal-cf/service-instance-reaper/arg"
	"github.com/pivotal-cf/service-instance-reaper/cloudfoundry"
	"github.com/pivotal-cf/service-instance-reaper/httpclient"
	reaperpkg "github.com/pivotal-cf/service-instance-reaper/reaper"
	"net/http"
	"os"
	"time"
)

func main() {
	username, password, skipSslValidation, reap, recursive, apiUrl, serviceName, planName, expiryInterval := arg.Parse(os.Args, os.Stdout, os.Exit)

	if !reap {
		fmt.Printf("DRY RUN ONLY!\n")
	}

	fmt.Printf("Reaping instances of the '%s' plan of '%s' older than %s in %s as %s...\n", planName, serviceName, durafmt.Parse(expiryInterval), apiUrl, username)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSslValidation},
	}
	client := &http.Client{Transport: transport}

	accessToken, err := cloudfoundry.GetOauthToken(client, apiUrl, username, password)
	if err != nil {
		fatalError("Authentication failed", err)
	}

	authClient := httpclient.NewAuthenticatedClient(client)
	cf := cloudfoundry.NewClient(authClient, apiUrl, accessToken)
	reaper := reaperpkg.NewReaper(cf, func() time.Time { return time.Now().UTC() }, os.Stdout)

	err = reaper.Reap(serviceName, planName, expiryInterval, reap, recursive)
	if err != nil {
		fatalError("Failed", err)
	}
}

func fatalError(message string, err error) {
	fmt.Printf("%s: %s", message, err)
	os.Exit(1)
}

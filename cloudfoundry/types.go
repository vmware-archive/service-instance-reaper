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

type Metadata struct {
	Guid      string
	CreatedAt string `json:"created_at"`
}

type Service struct {
	Metadata Metadata
}

type ServicePlan struct {
	Metadata Metadata
	Entity   struct {
		Name string
		Free bool
	}
}

type ServiceInstance struct {
	Metadata Metadata
	Entity   struct {
		Name string
	}
}

type listServicesResponse struct {
	Resources []Service
}

type listServicePlansResponse struct {
	NextUrl   string `json:"next_url"`
	Resources []ServicePlan
}

type listServicePlanInstancesResponse struct {
	NextUrl   string `json:"next_url"`
	Resources []ServiceInstance
}

type infoResponse struct {
	AuthorisationEndpoint string `json:"authorization_endpoint"`
}

type loginResponse struct {
	Links struct {
		Login string
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

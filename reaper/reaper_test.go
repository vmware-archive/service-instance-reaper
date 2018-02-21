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
package reaper_test

import (
	"errors"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/service-instance-reaper/cloudfoundry"
	"github.com/pivotal-cf/service-instance-reaper/cloudfoundry/cloudfoundryfakes"
	reaperpkg "github.com/pivotal-cf/service-instance-reaper/reaper"
	"time"
)

const (
	testServiceName                           = "test-service-name"
	testServiceGuid                           = "test-service-guid"
	testFreeServicePlanName                   = "test-free-service-name"
	testFreeServicePlanGuid                   = "test-free-service-guid"
	testSponsoredFreeServicePlanName          = "test-sponsored-free-service-name"
	testSponsoredFreeServicePlanGuid          = "test-sponsored-free-service-guid"
	testPaidServicePlanName                   = "test-paid-service-name"
	testPaidServicePlanGuid                   = "test-paid-service-guid"
	testExpiredFreePlanServiceInstanceGuid1   = "test-expired-free-plan-service-instance-guid-1"
	testExpiredFreePlanServiceInstanceName1   = "test-expired-free-plan-service-instance-name-1"
	testExpiredFreePlanServiceInstanceGuid2   = "test-expired-free-plan-service-instance-guid-2"
	testExpiredFreePlanServiceInstanceName2   = "test-expired-free-plan-service-instance-name-2"
	testNotExpiredFreePlanServiceInstanceGuid = "test-not-expired-free-plan-service-instance-guid"
	testNotExpiredFreePlanServiceInstanceName = "test-not-expired-free-plan-service-instance-name"
)

var _ = Describe("Reaper", func() {
	var (
		fakeCfClient       *cloudfoundryfakes.FakeClient
		expireAfter10Hours = 10 * time.Hour
		reap               = true
		recursive          = false
		testError          = errors.New("test error")
		reaper             reaperpkg.Reaper
		reaperOutput       *gbytes.Buffer
		reaperError        error
	)

	BeforeEach(func() {
		fakeCfClient = fakeCfClientFactory(
			successfulGetServicesResponse(),
			successfulGetServicePlansResponse(),
			successfulGetServicePlanInstancesResponse(),
		)
		reaperOutput = gbytes.NewBuffer()
		reap = true
		recursive = false
	})

	JustBeforeEach(func() {
		reaper = reaperpkg.NewReaper(fakeCfClient, frozenTime, reaperOutput)
		reaperError = reaper.Reap(testServiceName, testFreeServicePlanName, expireAfter10Hours, reap, recursive)
	})

	Describe("fetching services", func() {
		It("fetches a list of services with the given name", func() {
			Expect(reaperError).NotTo(HaveOccurred())
			Expect(fakeCfClient.GetServicesCallCount()).To(Equal(1), "Unexpected number of calls to GetServices")
			serviceName := fakeCfClient.GetServicesArgsForCall(0)
			Expect(serviceName).To(Equal(testServiceName))
		})

		Context("when fetching the list of services fails", func() {
			BeforeEach(func() {
				fakeCfClient.GetServicesReturns([]cloudfoundry.Service{}, testError)
			})

			It("logs the error and fails", func() {
				expectErrors(reaperError, reaperOutput, testError)
			})
		})

		Context("when the list of services is empty", func() {
			BeforeEach(func() {
				fakeCfClient.GetServicesReturns([]cloudfoundry.Service{}, nil)
			})

			It("reports that it has no work to do", func() {
				Expect(reaperError).NotTo(HaveOccurred())
				Expect(reaperOutput).To(gbytes.Say("No services of type '%s' found", testServiceName))
			})
		})
	})

	Describe("fetching service plans", func() {
		It("fetches a list of service plans for the service with the given guid", func() {
			Expect(reaperError).NotTo(HaveOccurred())
			Expect(fakeCfClient.GetServicePlansCallCount()).To(Equal(1), "Unexpected number of calls to GetServicePlans")
			serviceGuid := fakeCfClient.GetServicePlansArgsForCall(0)
			Expect(serviceGuid).To(Equal(testServiceGuid))
		})

		Context("when fetching the list of service plans fails", func() {
			BeforeEach(func() {
				fakeCfClient = fakeCfClientFactory(
					successfulGetServicesResponse(),
					servicePlanResult{nil, testError},
					successfulGetServicePlanInstancesResponse(),
				)
			})

			It("logs the error and fails", func() {
				expectErrors(reaperError, reaperOutput, testError)
			})
		})
	})

	Describe("fetching service plan instances", func() {
		It("fetches a list of instances of the given plan", func() {
			Expect(reaperError).NotTo(HaveOccurred())
			Expect(fakeCfClient.GetServicePlanInstancesCallCount()).To(Equal(1), "Unexpected number of calls to GetServicePlanInstances")
			servicePlanGuid := fakeCfClient.GetServicePlanInstancesArgsForCall(0)
			Expect(servicePlanGuid).To(Equal(testFreeServicePlanGuid))
		})

		Context("when fetching the list of service plan instances fails", func() {
			BeforeEach(func() {
				fakeCfClient = fakeCfClientFactory(
					successfulGetServicesResponse(),
					successfulGetServicePlansResponse(),
					serviceInstanceResult{nil, testError},
				)
			})

			It("logs the error and fails", func() {
				expectErrors(reaperError, reaperOutput, testError)
			})
		})
	})

	Describe("reaping", func() {
		Context("if the service plan instance response contains a malformed time string", func() {
			BeforeEach(func() {
				servicePlanInstancesResponseWithInvalidTime := successfulGetServicePlanInstancesResponse()
				servicePlanInstancesResponseWithInvalidTime.serviceInstances[0].Metadata.CreatedAt = "rubbish"

				fakeCfClient = fakeCfClientFactory(
					successfulGetServicesResponse(),
					successfulGetServicePlansResponse(),
					servicePlanInstancesResponseWithInvalidTime,
				)
			})

			It("logs the error and fails", func() {
				expectErrorsMatching(reaperError, reaperOutput, "invalid service instance creation time")
			})
		})

		Context("when the 'reap' flag is true", func() {
			BeforeEach(func() { reap = true })

			Context("when the 'recursive' flag is false", func() {
				BeforeEach(func() { recursive = false })

				It("deletes only the expired service instances", func() {
					Expect(reaperError).NotTo(HaveOccurred())

					Expect(fakeCfClient.DeleteServiceInstanceCallCount()).To(Equal(2), "Unexpected number of DeleteServiceInstance invocations")

					deletedServiceInstanceGuid, deletedRecursively := fakeCfClient.DeleteServiceInstanceArgsForCall(0)
					Expect(deletedServiceInstanceGuid).To(Equal(testExpiredFreePlanServiceInstanceGuid1))
					Expect(deletedRecursively).To(BeFalse())

					deletedServiceInstanceGuid, deletedRecursively = fakeCfClient.DeleteServiceInstanceArgsForCall(1)
					Expect(deletedServiceInstanceGuid).To(Equal(testExpiredFreePlanServiceInstanceGuid2))
					Expect(deletedRecursively).To(BeFalse())
				})
			})

			Context("when the 'recursive' flag is true", func() {
				BeforeEach(func() { recursive = true })

				It("deletes only the expired service instances, recursively", func() {
					Expect(reaperError).NotTo(HaveOccurred())

					Expect(fakeCfClient.DeleteServiceInstanceCallCount()).To(Equal(2), "Unexpected number of DeleteServiceInstance invocations")

					deletedServiceInstanceGuid, deletedRecursively := fakeCfClient.DeleteServiceInstanceArgsForCall(0)
					Expect(deletedServiceInstanceGuid).To(Equal(testExpiredFreePlanServiceInstanceGuid1))
					Expect(deletedRecursively).To(BeTrue())

					deletedServiceInstanceGuid, _ = fakeCfClient.DeleteServiceInstanceArgsForCall(1)
					Expect(deletedServiceInstanceGuid).To(Equal(testExpiredFreePlanServiceInstanceGuid2))
					Expect(deletedRecursively).To(BeTrue())
				})
			})

			Context("when service instance deletion fails", func() {
				BeforeEach(func() {
					fakeCfClient.DeleteServiceInstanceReturns(testError)
				})

				It("logs the error and fails", func() {
					const errorMessage = "unable to delete service instance: %s %s \\(%s\\)\n"
					expectErrorsMatching(reaperError, reaperOutput,
						fmt.Sprintf(errorMessage, testExpiredFreePlanServiceInstanceName1, testExpiredFreePlanServiceInstanceGuid1, testError),
						fmt.Sprintf(errorMessage, testExpiredFreePlanServiceInstanceName2, testExpiredFreePlanServiceInstanceGuid2, testError),
					)
				})
			})
		})

		Context("when the 'reap' flag is false", func() {
			BeforeEach(func() { reap = false })

			It("logs only the expired service instance names and guids", func() {
				Expect(reaperError).NotTo(HaveOccurred())
				Expect(reaperOutput).To(gbytes.Say("%s %s\n", testExpiredFreePlanServiceInstanceName1, testExpiredFreePlanServiceInstanceGuid1))
				Expect(reaperOutput).To(gbytes.Say("%s %s\n", testExpiredFreePlanServiceInstanceName2, testExpiredFreePlanServiceInstanceGuid2))
				Expect(reaperOutput).NotTo(gbytes.Say("%s %s\n", testNotExpiredFreePlanServiceInstanceName, testNotExpiredFreePlanServiceInstanceGuid))
				Expect(fakeCfClient.DeleteServiceInstanceCallCount()).To(Equal(0), "Unexpected call to DeleteServiceInstance!")
			})
		})
	})
})

type servicePlanResult struct {
	servicePlans []cloudfoundry.ServicePlan
	err          error
}

type serviceInstanceResult struct {
	serviceInstances []cloudfoundry.ServiceInstance
	err              error
}

func expectErrorsMatching(reaperError error, output *gbytes.Buffer, expectedErrorMessages ...string) {
	for _, errorMessage := range expectedErrorMessages {
		Expect(output).To(gbytes.Say(errorMessage))
	}
	Expect(reaperError).To(HaveOccurred())
}

func expectErrors(reaperErrors error, output *gbytes.Buffer, expectedErrors ...error) {
	expectedErrorMessages := make([]string, len(expectedErrors))
	for i, err := range expectedErrors {
		expectedErrorMessages[i] = err.Error()
	}
	expectErrorsMatching(reaperErrors, output, expectedErrorMessages...)
}

func fakeCfClientFactory(services []cloudfoundry.Service, servicePlans servicePlanResult, serviceInstances serviceInstanceResult) *cloudfoundryfakes.FakeClient {
	cf := &cloudfoundryfakes.FakeClient{}

	cf.GetServicesReturns(services, nil)

	cf.GetServicePlansReturns(servicePlans.servicePlans, servicePlans.err)

	serviceInstancesChannel := make(chan cloudfoundry.ServiceInstance, len(serviceInstances.serviceInstances))
	serviceInstanceErrorsChannel := make(chan error, 1)
	defer func() { close(serviceInstancesChannel) }()
	defer func() { close(serviceInstanceErrorsChannel) }()
	for _, serviceInstance := range serviceInstances.serviceInstances {
		serviceInstancesChannel <- serviceInstance
	}
	if serviceInstances.err != nil {
		serviceInstanceErrorsChannel <- serviceInstances.err
	}
	cf.GetServicePlanInstancesReturns(serviceInstancesChannel, serviceInstanceErrorsChannel)

	return cf
}

func successfulGetServicesResponse() []cloudfoundry.Service {
	return []cloudfoundry.Service{
		{cloudfoundry.Metadata{Guid: testServiceGuid}},
	}
}

func successfulGetServicePlansResponse() servicePlanResult {
	return servicePlanResult{
		servicePlans: []cloudfoundry.ServicePlan{
			{
				cloudfoundry.Metadata{Guid: testPaidServicePlanGuid},
				struct {
					Name string
					Free bool
				}{testPaidServicePlanName, false},
			},
			{
				cloudfoundry.Metadata{Guid: testFreeServicePlanGuid},
				struct {
					Name string
					Free bool
				}{testFreeServicePlanName, true},
			},
			{
				cloudfoundry.Metadata{Guid: testSponsoredFreeServicePlanGuid},
				struct {
					Name string
					Free bool
				}{testSponsoredFreeServicePlanName, true},
			},
		},
		err: nil,
	}
}

func successfulGetServicePlanInstancesResponse() serviceInstanceResult {
	return serviceInstanceResult{
		serviceInstances: []cloudfoundry.ServiceInstance{
			{
				cloudfoundry.Metadata{
					Guid:      testExpiredFreePlanServiceInstanceGuid1,
					CreatedAt: fifteenHoursAgo().Format(time.RFC3339),
				},
				struct {
					Name string
				}{testExpiredFreePlanServiceInstanceName1},
			},
			{
				cloudfoundry.Metadata{
					Guid:      testExpiredFreePlanServiceInstanceGuid2,
					CreatedAt: tenHoursOneSecondAgo().Format(time.RFC3339),
				},
				struct {
					Name string
				}{testExpiredFreePlanServiceInstanceName2},
			},
			{
				cloudfoundry.Metadata{
					Guid:      testNotExpiredFreePlanServiceInstanceGuid,
					CreatedAt: tenHoursAgo().Format(time.RFC3339),
				},
				struct {
					Name string
				}{testNotExpiredFreePlanServiceInstanceName},
			},
		},
		err: nil,
	}
}

func frozenTime() time.Time {
	return time.Date(2018, 1, 24, 20, 00, 0, 0, time.UTC)
}

func tenHoursAgo() time.Time {
	return frozenTime().Add(-10 * time.Hour)
}

func tenHoursOneSecondAgo() time.Time {
	return frozenTime().Add(-10 * time.Hour).Add(-1 * time.Second)
}

func fifteenHoursAgo() time.Time {
	return frozenTime().Add(-15 * time.Hour)
}

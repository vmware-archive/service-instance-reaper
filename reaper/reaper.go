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
	"errors"
	"fmt"
	"github.com/pivotal-cf/service-instance-reaper/cloudfoundry"
	"io"
	"time"
)

type Reaper struct {
	cf             cloudfoundry.Client
	expiryInterval time.Duration
	reap           bool
	recursive      bool
	currentTime    func() time.Time
	output         io.Writer
	errorChannel   chan error
}

func NewReaper(cf cloudfoundry.Client, currentTime func() time.Time, output io.Writer) Reaper {
	return Reaper{
		cf:          cf,
		currentTime: currentTime,
		output:      output,
	}
}

func (r Reaper) Reap(serviceName string, expiryInterval time.Duration, reap, recursive bool) error {
	r.expiryInterval = expiryInterval
	r.reap = reap
	r.recursive = recursive
	r.errorChannel = make(chan error, 100)

	r.delete(r.expiredInstancesOf(r.freeServicePlansOf(r.servicesWithName(serviceName))))

	errorsFound := false
	for err := range r.errorChannel {
		errorsFound = true
		fmt.Fprintln(r.output, err)
	}

	if errorsFound {
		return errors.New("errors occurred whilst reaping")
	}

	return nil
}

func (r *Reaper) servicesWithName(serviceName string) <-chan cloudfoundry.Service {
	output := make(chan cloudfoundry.Service, 1)

	go func() {
		defer close(output)

		services, err := r.cf.GetServices(serviceName)
		if err != nil {
			r.errorChannel <- err
			return
		}

		if len(services) == 0 {
			fmt.Fprintf(r.output, "No services of type '%s' found", serviceName)
			return
		}

		output <- services[0]
	}()

	return output
}

func (r *Reaper) freeServicePlansOf(services <-chan cloudfoundry.Service) <-chan cloudfoundry.ServicePlan {
	output := make(chan cloudfoundry.ServicePlan, cloudfoundry.MaximumResultsPerPage)

	go func() {
		defer close(output)

		for service := range services {
			servicePlans, err := r.cf.GetServicePlans(service.Metadata.Guid)
			if err != nil {
				r.errorChannel <- err
				return
			}

			for _, servicePlan := range servicePlans {
				if servicePlan.Entity.Free {
					output <- servicePlan
				}
			}
		}
	}()

	return output
}

func (r *Reaper) expiredInstancesOf(servicePlans <-chan cloudfoundry.ServicePlan) <-chan cloudfoundry.ServiceInstance {
	output := make(chan cloudfoundry.ServiceInstance, cloudfoundry.MaximumResultsPerPage)

	go func() {
		defer close(output)

		for servicePlan := range servicePlans {
			serviceInstances, serviceInstanceErrors := r.cf.GetServicePlanInstances(servicePlan.Metadata.Guid)

			for serviceInstance := range serviceInstances {
				serviceInstanceExpired, err := expired(serviceInstance.Metadata.CreatedAt, r.expiryInterval, r.currentTime)
				if err != nil {
					r.errorChannel <- err
					return
				}

				if serviceInstanceExpired {
					output <- serviceInstance
				}
			}

			mergeErrors(serviceInstanceErrors, r.errorChannel)
		}
	}()

	return output
}

func (r *Reaper) delete(serviceInstances <-chan cloudfoundry.ServiceInstance) {
	go func() {
		defer close(r.errorChannel)

		for serviceInstance := range serviceInstances {
			if r.reap {
				err := r.cf.DeleteServiceInstance(serviceInstance.Metadata.Guid, r.recursive)
				if err != nil {
					r.errorChannel <- fmt.Errorf("unable to delete service instance: %s %s (%s)\n",
						serviceInstance.Entity.Name, serviceInstance.Metadata.Guid, err)
				}
			}

			fmt.Fprintf(r.output, "%s %s\n", serviceInstance.Entity.Name, serviceInstance.Metadata.Guid)
		}
	}()
}

func mergeErrors(from <-chan error, to chan<- error) {
	for err := range from {
		to <- err
	}
}

func expired(creationTimeString string, expiryInterval time.Duration, currentTime func() time.Time) (bool, error) {
	creationTime, err := time.Parse(time.RFC3339, creationTimeString)
	if err != nil {
		return false, fmt.Errorf("invalid service instance creation time: %s", err)
	}
	expiryTime := creationTime.Add(expiryInterval)
	return currentTime().After(expiryTime), nil
}

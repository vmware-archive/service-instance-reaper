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
package arg_test

import (
	"github.com/pivotal-cf/service-instance-reaper/arg"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Parse", func() {

	var (
		args              []string
		username          string
		password          string
		skipSslValidation bool
		reap              bool
		apiUrl            string
		serviceName       string
		expiryInterval    time.Duration
		failed bool
		exitCode          int
	)

	JustBeforeEach(func() {
		failed = false
		username, password, skipSslValidation, reap, apiUrl, serviceName, expiryInterval = arg.Parse(args, func(code int) { failed = true; exitCode = code })
	})

	Context("with a full set of arguments", func() {
		BeforeEach(func() {
			args = []string{"command", "-u=user", "-p=password", "-skip-ssl-validation", "-reap", "some.url", "service-instance", "168"}
		})

		It("should not fail", func() {
		    Expect(failed).To(BeFalse())
		})

		It("should parse the arguments correctly", func() {
		    Expect(username).To(Equal("user"))
		    Expect(password).To(Equal("password"))
		    Expect(skipSslValidation).To(BeTrue())
		    Expect(reap).To(BeTrue())
		    Expect(apiUrl).To(Equal("https://some.url"))
		    Expect(serviceName).To(Equal("service-instance"))
		    Expect(expiryInterval).To(Equal(time.Duration(168)*time.Hour))
		})
	})

	Context("with a minimal set of arguments", func() {
		BeforeEach(func() {
			args = []string{"command", "-u=user", "-p=password", "some.url", "service-instance", "168"}
		})

		It("should not fail", func() {
		    Expect(failed).To(BeFalse())
		})

		It("should apply the correct defaults", func() {
		    Expect(skipSslValidation).To(BeFalse())
		    Expect(reap).To(BeFalse())
		})


		It("should still parse the specified arguments correctly", func() {
			Expect(username).To(Equal("user"))
			Expect(password).To(Equal("password"))
			Expect(apiUrl).To(Equal("https://some.url"))
			Expect(serviceName).To(Equal("service-instance"))
			Expect(expiryInterval).To(Equal(time.Duration(168)*time.Hour))
		})
	})

	Context("with an invalid number of arguments", func() {
		BeforeEach(func() {
			// Pass an additional argument so that parsing will not fail after the failure closure returns
			args = []string{"command", "-u=user", "-p=password", "some.url", "service-instance", "168", "banana"}
		})

		It("should fail with exit status code 0", func() {
			Expect(failed).To(BeTrue())
			Expect(exitCode).To(Equal(0))
		})
	})

	Context("when help is requested", func() {
		BeforeEach(func() {
			// Pass other arguments in addition to "help" so that parsing will not fail after the failure closure returns
			args = []string{"command", "-u=user", "-p=password", "help", "service-instance", "168"}
		})

		It("should fail with exit status code 0", func() {
			Expect(failed).To(BeTrue())
			Expect(exitCode).To(Equal(0))
		})
	})

	Context("when an invalid expiry interval is specified", func() {
		BeforeEach(func() {
			// Pass other arguments in addition to "help" so that parsing will not fail after the failure closure returns
			args = []string{"command", "-u=user", "-p=password", "some.url", "service-instance", "-1"}
		})

		It("should fail with exit status code 0", func() {
			Expect(failed).To(BeTrue())
			Expect(exitCode).To(Equal(1))
		})
	})
})

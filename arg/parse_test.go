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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/service-instance-reaper/arg"
	"time"
)

var _ = Describe("Parse", func() {

	const (
		testServiceName    = "p-config-server"
		testPlanName       = "planName"
		testUrl            = "some.url"
		expirationInterval = "168"
	)

	var (
		args              []string
		username          string
		password          string
		planName          string
		skipSslValidation bool
		reap              bool
		recursive         bool
		apiUrl            string
		serviceName       string
		expiryInterval    time.Duration
		shouldExit        bool
		exitCode          int
		output            *gbytes.Buffer
	)

	JustBeforeEach(func() {
		shouldExit = false
		output = gbytes.NewBuffer()
		username, password, skipSslValidation, reap, recursive, apiUrl, serviceName, planName, expiryInterval = arg.Parse(args, output, func(code int) { shouldExit = true; exitCode = code })
	})

	Context("with a full set of arguments", func() {
		BeforeEach(func() {
			args = []string{"command", "-u=user", "-p=password", "-skip-ssl-validation", "-reap", "-recursive", testUrl, testServiceName, testPlanName, expirationInterval}
		})

		It("does not fail", func() {
			Expect(shouldExit).To(BeFalse())
		})

		It("parses the arguments correctly", func() {
			Expect(username).To(Equal("user"))
			Expect(password).To(Equal("password"))
			Expect(skipSslValidation).To(BeTrue())
			Expect(reap).To(BeTrue())
			Expect(recursive).To(BeTrue())
			Expect(apiUrl).To(Equal("https://some.url"))
			Expect(serviceName).To(Equal("p-config-server"))
			Expect(planName).To(Equal("planName"))
			Expect(expiryInterval).To(Equal(time.Duration(168) * time.Hour))
		})
	})

	Context("with a minimal set of arguments", func() {
		BeforeEach(func() {
			args = []string{"command", "-u=user", "-p=password", testUrl, testServiceName, testPlanName, expirationInterval}
		})

		It("does not fail", func() {
			Expect(shouldExit).To(BeFalse())
		})

		It("applies the correct defaults", func() {
			Expect(skipSslValidation).To(BeFalse())
			Expect(reap).To(BeFalse())
			Expect(recursive).To(BeFalse())
		})

		It("parses the specified arguments correctly", func() {
			Expect(username).To(Equal("user"))
			Expect(password).To(Equal("password"))
			Expect(apiUrl).To(Equal("https://some.url"))
			Expect(serviceName).To(Equal("p-config-server"))
			Expect(planName).To(Equal("planName"))
			Expect(expiryInterval).To(Equal(time.Duration(168) * time.Hour))
		})
	})

	Context("with an invalid number of arguments", func() {
		BeforeEach(func() {
			// Pass an additional argument so that parsing will not fail after the failure closure returns
			args = []string{"command", "-u=user", "-p=password", testUrl, testServiceName, testPlanName, expirationInterval, "banana"}
		})

		It("fails with exit status code 0", func() {
			Expect(shouldExit).To(BeTrue())
			Expect(exitCode).To(Equal(0))
		})

		It("prints usage information", func() {
			Expect(output).To(gbytes.Say("Usage"))
		})
	})

	Context("when help is requested", func() {
		BeforeEach(func() {
			args = []string{"command", "-u=user", "-p=password", "help"}
		})

		It("fails with exit status code 0", func() {
			Expect(shouldExit).To(BeTrue())
			Expect(exitCode).To(Equal(0))
		})

		It("prints usage information", func() {
			Expect(output).To(gbytes.Say("Usage"))
		})
	})

	Context("when an invalid api url is specified", func() {
		BeforeEach(func() {
			args = []string{"command", "-u=user", "-p=password", ":///:/", testServiceName, testPlanName, expirationInterval}
		})

		It("fails with exit status code 1", func() {
			Expect(shouldExit).To(BeTrue())
			Expect(exitCode).To(Equal(1))
		})

		It("prints usage information", func() {
			Expect(output).To(gbytes.Say("Usage"))
		})
	})

	Context("when an invalid expiry interval is specified", func() {
		BeforeEach(func() {
			args = []string{"command", "-u=user", "-p=password", testUrl, testServiceName, testPlanName, "-1"}
		})

		It("fails with exit status code 0", func() {
			Expect(shouldExit).To(BeTrue())
			Expect(exitCode).To(Equal(1))
		})
	})
})

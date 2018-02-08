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
package arg

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"
)

func Parse(args []string, output io.Writer, exit func(int)) (username string, password string, skipSslValidation bool, reap, recursive bool, apiUrl string, serviceName string, expiryInterval time.Duration) {
	commandLine := flag.NewFlagSet(args[0], flag.ExitOnError)
	commandLine.SetOutput(output)
	commandLine.StringVar(&username, "u", "", "username")
	commandLine.StringVar(&password, "p", "", "password")
	commandLine.BoolVar(&skipSslValidation, "skip-ssl-validation", false, "Skip verification of the API endpoint. Not recommended!")
	commandLine.BoolVar(&reap, "reap", false, "Reap service instances. Otherwise perform a dry run only.")
	commandLine.BoolVar(&recursive, "recursive", false, "Also deletes any service bindings, service keys, and routes associated with reaped service instances.")
	commandLine.Parse(args[1:])

	positionalArgs := commandLine.Args()
	if len(positionalArgs) != 3 || positionalArgs[0] == "help" {
		printUsage(output, commandLine)
		exit(0)
		return
	}

	urlArg, err := url.Parse(positionalArgs[0])
	if err != nil {
		fmt.Fprintf(output, "Invalid api url: %s\n", positionalArgs[0])
		printUsage(output, commandLine)
		exit(1)
		return
	}
	urlArg.Scheme = "https"
	apiUrl = urlArg.String()

	serviceName = positionalArgs[1]

	expiryIntervalHours, err := strconv.ParseFloat(positionalArgs[2], 32)
	if err != nil || expiryIntervalHours < 0 {
		fmt.Fprintf(output, "Invalid expiry interval: %s\n", positionalArgs[2])
		printUsage(output, commandLine)
		exit(1)
		return
	}
	expiryInterval = time.Duration(expiryIntervalHours*60*60) * time.Second

	return
}

func printUsage(output io.Writer, flags *flag.FlagSet) {
	fmt.Fprintln(output, `Delete instances of the given service older than the given age
		
Usage:
  service-instance-reaper [-reap] [-recursive] -u username -p password [-skip-ssl-validation] API_URL SERVICE_NAME AGE_HOURS

Flags (which must be specified BEFORE non-flag arguments):`)
	flags.PrintDefaults()
}

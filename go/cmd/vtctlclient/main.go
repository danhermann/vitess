/*
Copyright 2019 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"vitess.io/vitess/go/exit"
	"vitess.io/vitess/go/trace"
	"vitess.io/vitess/go/vt/log"
	"vitess.io/vitess/go/vt/logutil"
	"vitess.io/vitess/go/vt/vtctl/vtctlclient"

	logutilpb "vitess.io/vitess/go/vt/proto/logutil"

	// Include deprecation warnings for soon-to-be-unsupported flag invocations.
	_flag "vitess.io/vitess/go/internal/flag"
)

// The default values used by these flags cannot be taken from wrangler and
// actionnode modules, as we don't want to depend on them at all.
var (
	actionTimeout = flag.Duration("action_timeout", time.Hour, "timeout for the total command")
	server        = flag.String("server", "", "server to use for connection")
)

// checkDeprecations runs quick and dirty checks to see whether any command or flag are deprecated.
// For any depracated command or flag, the function issues a warning message.
// this function will change on each Vitess version. Each depracation message should only last a version.
// VEP-4 will replace the need for this function. See https://github.com/vitessio/enhancements/blob/main/veps/vep-4.md
func checkDeprecations(args []string) {
	// utility:
	findSubstring := func(s string) (arg string, ok bool) {
		for _, arg := range args {
			if strings.Contains(arg, s) {
				return arg, true
			}
		}
		return "", false
	}
	if _, ok := findSubstring("ApplySchema"); ok {
		if arg, ok := findSubstring("ddl_strategy"); ok {
			if strings.Contains(arg, "-skip-topo") {
				log.Warning("-skip-topo is deprecated and will be removed in future versions")
			}
		}
	}
}

func main() {
	defer exit.Recover()

	_flag.Parse()

	closer := trace.StartTracing("vtctlclient")
	defer trace.LogErrorsWhenClosing(closer)

	logger := logutil.NewConsoleLogger()

	// We can't do much without a --server flag
	if *server == "" {
		log.Error(errors.New("please specify --server <vtctld_host:vtctld_port> to specify the vtctld server to connect to"))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *actionTimeout)
	defer cancel()

	checkDeprecations(flag.Args())

	err := vtctlclient.RunCommandAndWait(
		ctx, *server, _flag.Args(),
		func(e *logutilpb.Event) {
			logutil.LogEvent(logger, e)
		})
	if err != nil {
		if strings.Contains(err.Error(), "flag: help requested") {
			return
		}

		errStr := strings.Replace(err.Error(), "remote error: ", "", -1)
		fmt.Printf("%s Error: %s\n", _flag.Arg(0), errStr)
		log.Error(err)
		os.Exit(1)
	}
}

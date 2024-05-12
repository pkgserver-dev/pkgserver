/*
Copyright 2024 Nokia.

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
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/pkgserver-dev/pkgserver/cmd/pkgctl/commands"
	"github.com/henderiw/logger/log"
)

const (
	//defaultConfigFileSubDir = "pkgctl"
	//defaultConfigFileName   = "pkgctl.yaml"
)

func main() {
	os.Exit(runMain())
}

// runMain does the initial setup to setup logging
func runMain() int {
	// init logging
	l := log.NewLogger(&log.HandlerOptions{Name: "pkgctl-logger", AddSource: false})
	slog.SetDefault(l)

	// init context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx = log.IntoContext(ctx, l)

	// init cmd context
	cmd := commands.GetMain(ctx)

	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s \n", err.Error())
		cancel()
		return 1
	}
	return 0
}
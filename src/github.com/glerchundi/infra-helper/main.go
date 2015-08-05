// Copyright (c) 2015 Gorka Lerchundi Osa. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
package main

import (
	"github.com/codegangsta/cli"
	"github.com/glerchundi/infra-helper/command"
)

func main() {
	app := cli.NewApp()
	app.Name = "infra-helper"
	app.Version = "0.1.1"
	app.Usage = "manage etcd cluster based on AWS autoscaling groups"
	app.Commands = []cli.Command{
		command.NewSyncEtcdPeersCommand(),
		command.NewListAutoscaleMembersCommand(),
	}
	app.RunAndExitOnError()
}
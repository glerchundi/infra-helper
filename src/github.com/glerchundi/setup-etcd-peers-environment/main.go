// Copyright (c) 2015 Gorka Lerchundi Osa. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/coreos/etcd/client"
	"github.com/glerchundi/setup-etcd-peers-environment/providers"
	"github.com/glerchundi/setup-etcd-peers-environment/providers/aws"
	"github.com/glerchundi/setup-etcd-peers-environment/util"
)

func main() {
	app := cli.NewApp()
	app.Name = "setup-etcd-peers-environment"
	app.Version = "0.1.1"
	app.Usage = "manage etcd cluster peers based on AWS autoscaling groups"
	app.Action = mainAction
	app.Flags = []cli.Flag {
		cli.StringFlag{
			Name: "out, o",
			Value: "/etc/sysconfig/etcd-peers",
			Usage: "etcd peers environment config file destination",
		},
	}
	app.RunAndExitOnError()
}

func mainAction(c *cli.Context) {
	environmentFilePath := c.GlobalString("out")
	if _, err := os.Stat(environmentFilePath); err == nil {
		log.Printf("etcd-peers file %s already created, exiting.", environmentFilePath)
    	return
	}

	tempFilePath := environmentFilePath + ".tmp"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		log.Fatal(err)
	}

	defer tempFile.Close()
	if err := writeEnvironment(tempFile); err != nil {
		log.Fatal(err)
	}

	if err := os.Rename(tempFilePath, environmentFilePath); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

func writeEnvironment(w io.Writer) error {
	var buffer bytes.Buffer
	var err error
	var provider providers.Provider = aws.New()

	instanceId, err := provider.GetInstanceId()
	if err != nil {
		return err
	}

	instanceIp, err := provider.GetInstancePrivateAddress()
	if err != nil {
		return err
	}

	clusterMembersByName, err := provider.GetClusterMembers()
	if err != nil {
		return err
	}

	// retrieve current cluster members
	var etcdMembers []client.Member
	var goodEtcdClientURL string = ""
	for _, memberIp := range clusterMembersByName {
		if memberIp == instanceIp {
			continue
		}

		etcdClientURL := util.EtcdClientURLFromIP(memberIp)
		etcdMembers, err = util.EtcdListMembers(etcdClientURL)
		if err == nil {
			goodEtcdClientURL = etcdClientURL
			break
		}
	}

	// etcd parameters
	var initialClusterState string
    var initialCluster string

	// check if instanceId is already member of cluster
	var isMember bool = false
	for _, member := range etcdMembers {
		if member.Name == instanceId {
			isMember = true
			break
		}
	}

	// if i am not already listed as a member of the cluster assume that this is a new cluster
	if etcdMembers != nil && !isMember {
		log.Print("joining an existing cluster, using this client url: ", goodEtcdClientURL)
		initialClusterState = "existing"

		//
		// detect and remove bad peers
		//

		// create a reverse cluster members map
		clusterMembersByIp := make(map[string]string)
		for memberName, memberIp := range clusterMembersByName {
			clusterMembersByIp[memberIp] = memberName
		}

		for _, etcdMember := range etcdMembers {
			peerURL, err := url.Parse(etcdMember.PeerURLs[0])
			if err != nil {
				return err
			}

			peerHost, _, err := net.SplitHostPort(peerURL.Host)
			if err != nil {
				return err
			}

			if _, ok := clusterMembersByIp[peerHost]; !ok {
				log.Print("removing etcd member ", etcdMember.ID)
				err = util.EtcdRemoveMember(goodEtcdClientURL, etcdMember.ID)
				if err != nil {
					return err
				}
			}
		}

		//
		// join an existing cluster
		//

		instancePeerURL := util.EtcdPeerURLFromIP(instanceIp)
		log.Print("adding etcd member ", instancePeerURL)
		member, err := util.EtcdAddMember(goodEtcdClientURL, instancePeerURL)
		if member != nil {
			return err
		}
	} else {
		log.Print("creating new cluster")
		initialClusterState = "new"
	}

	// initial cluster
	kvs := make([]string, 0)
	for memberName, memberIp := range clusterMembersByName {
		kvs = append(kvs, fmt.Sprintf("%s=%s", memberName, util.EtcdPeerURLFromIP(memberIp)))
	}
	initialCluster = strings.Join(kvs, ",")

	// create environment variables
	buffer.WriteString(fmt.Sprintf("ETCD_NAME=%s\n", instanceId))
	buffer.WriteString(fmt.Sprintf("ETCD_INITIAL_CLUSTER_STATE=%s\n", initialClusterState))
	buffer.WriteString(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s\n", initialCluster))

	if _, err := buffer.WriteTo(w); err != nil {
		return err
	}
	
	return nil
}
package command

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
	"github.com/glerchundi/infra-helper/providers"
	"github.com/glerchundi/infra-helper/providers/aws"
	"github.com/glerchundi/infra-helper/util"
)

func NewSyncEtcdPeersCommand() cli.Command {
	return cli.Command{
		Name:  "sync-etcd-peers",
		Usage: `syncs "etcd" cluster (adds/removes members based on 'autoscale' information)`,
		Flags: []cli.Flag {
			cli.StringFlag{
				Name: "out, o",
				Value: "/etc/sysconfig/etcd-peers",
				Usage: "etcd peers environment file destination",
			},
		},
		Action: handleSyncEtcdPeers,
	}
}

func handleSyncEtcdPeers(c *cli.Context) {
	environmentFilePath := c.String("out")
	if _, err := os.Stat(environmentFilePath); err == nil {
		log.Printf("etcd-peers file %s already created, exiting.\n", environmentFilePath)
		return
	}

	tempFilePath := environmentFilePath + ".tmp"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		log.Fatal(err)
	}

	isClosed := false
	defer func() { if !isClosed { tempFile.Close() } }()

	if err := writeEnvironment(tempFile); err != nil {
		log.Fatal(err)
	}

	if err := tempFile.Close(); err != nil {
		log.Fatal(err)
	}

	isClosed = true

	if err := os.Rename(tempFilePath, environmentFilePath); err != nil {
		log.Fatal(err)
	}

	return
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
		log.Printf("joining to an existing cluster, using this client url: %s\n", goodEtcdClientURL)

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
				log.Printf("removing etcd member: %s...", etcdMember.ID)
				err = util.EtcdRemoveMember(goodEtcdClientURL, etcdMember.ID)
				if err != nil {
					return err
				}
				log.Printf("done\n")
			}
		}

		//
		// list current etcd members (after removing spurious ones)
		//

		etcdMembers, err = util.EtcdListMembers(goodEtcdClientURL)
		if err != nil {
			return err
		}

		kvs := make([]string, 0)
		for _, etcdMember := range etcdMembers {
			// ignore unstarted peers
			if len(etcdMember.Name) == 0 {
				continue
			}
			kvs = append(kvs, fmt.Sprintf("%s=%s", etcdMember.Name, etcdMember.PeerURLs[0]))
		}
		kvs = append(kvs, fmt.Sprintf("%s=%s", instanceId, util.EtcdPeerURLFromIP(instanceIp)))

		initialClusterState = "existing"
		initialCluster = strings.Join(kvs, ",")

		//
		// join an existing cluster
		//

		instancePeerURL := util.EtcdPeerURLFromIP(instanceIp)
		log.Printf("adding etcd member: %s...", instancePeerURL)
		member, err := util.EtcdAddMember(goodEtcdClientURL, instancePeerURL)
		if member == nil {
			return err
		}
		log.Printf("done\n")
	} else {
		log.Printf("creating new cluster\n")

		// initial cluster
		kvs := make([]string, 0)
		for memberName, memberIp := range clusterMembersByName {
			kvs = append(kvs, fmt.Sprintf("%s=%s", memberName, util.EtcdPeerURLFromIP(memberIp)))
		}

		initialClusterState = "new"
		initialCluster = strings.Join(kvs, ",")
	}

	// indicate it's going to write envvars
	log.Printf("writing environment variables...")

	// create environment variables
	buffer.WriteString(fmt.Sprintf("ETCD_NAME=%s\n", instanceId))
	buffer.WriteString(fmt.Sprintf("ETCD_INITIAL_CLUSTER_STATE=%s\n", initialClusterState))
	buffer.WriteString(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s\n", initialCluster))

	if _, err := buffer.WriteTo(w); err != nil {
		return err
	}

	// write done
	log.Printf("done\n")

	return nil
}
// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Modified by: Gorka Lerchundi Osa

package util

import (
	"fmt"
	"net/http"
	"os"

	"github.com/coreos/etcd/client"
	"github.com/coreos/etcd/pkg/transport"
	"golang.org/x/net/context"
)

func getEtcdTransport() (*http.Transport, error) {
	return transport.NewTransport(
		transport.TLSInfo{
			CAFile:   os.Getenv("ETCDCTL_CA_FILE"),
			CertFile: os.Getenv("ETCDCTL_CERT_FILE"),
			KeyFile:  os.Getenv("ETCDCTL_KEY_FILE"),
		},
	)
}

func newEtcdClient(url string) (client.Client, error) {
	tr, err := getEtcdTransport()
	if err != nil {
		return nil, err
	}

	cfg := client.Config{
		Transport: tr,
		Endpoints: []string{url},
	}

	hc, err := client.New(cfg)
	if err != nil {
		return nil, err
	}

	return hc, nil
}

func newEtcdMembersAPI(url string) (client.MembersAPI, error) {
	hc, err := newEtcdClient(url)
	if err != nil {
		return nil, err
	}

	return client.NewMembersAPI(hc), nil
}

func EtcdPeerURLFromIP(ip string) string {
	return fmt.Sprintf("http://%s:2380", ip)
}

func EtcdClientURLFromIP(ip string) string {
	return fmt.Sprintf("http://%s:2379", ip)
}

func EtcdListMembers(url string) (members []client.Member, err error) {
	mAPI, err := newEtcdMembersAPI(url)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), client.DefaultRequestTimeout)
	members, err = mAPI.List(ctx)
	cancel()

	return
}

func EtcdAddMember(url string, peerURL string) (member *client.Member, err error) {
	mAPI, err := newEtcdMembersAPI(url)
	if err != nil {
		return nil, err
	}

	// Actually attempt to remove the member.
	ctx, cancel := context.WithTimeout(context.Background(), client.DefaultRequestTimeout)
	member, err = mAPI.Add(ctx, peerURL)
	cancel()

	return
}

func EtcdRemoveMember(url, removalID string) (err error) {
	mAPI, err := newEtcdMembersAPI(url)
	if err != nil {
		return err
	}

	// Actually attempt to remove the member.
	ctx, cancel := context.WithTimeout(context.Background(), client.DefaultRequestTimeout)
	err = mAPI.Remove(ctx, removalID)
	cancel()

	return
}
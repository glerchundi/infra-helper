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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"

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

// because API doesn't permit adding 'name' as well as 'clientURLs' we use it directly bypassing
// go library.

// v2MembersURL add the necessary path to the provided endpoint
// to route requests to the default v2 members API.
func v2MembersURL(ep url.URL) *url.URL {
	ep.Path = path.Join(ep.Path, "/v2/members")
	return &ep
}

func assertStatusCode(got int, want ...int) (err error) {
	for _, w := range want {
		if w == got {
			return nil
		}
	}
	return fmt.Errorf("unexpected status code %d", got)
}

type membersAPIActionAdd struct {
	Name string
	PeerURL string
	ClientURL string
}

func (m *membersAPIActionAdd) MarshalJSON() ([]byte, error) {
	s := struct {
		Name string `json:"name"`
		PeerURLs []string `json:"peerURLs"`
		ClientURLs []string `json:"clientURLs"`
	}{
		Name: m.Name,
		PeerURLs: []string{m.PeerURL},
		ClientURLs: []string{m.ClientURL},
	}

	return json.Marshal(&s)
}

func (a *membersAPIActionAdd) HTTPRequest(ep url.URL) *http.Request {
	u := v2MembersURL(ep)
	b, _ := json.Marshal(a)
	req, _ := http.NewRequest("POST", u.String(), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
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

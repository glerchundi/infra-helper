// Copyright (c) 2015 Gorka Lerchundi Osa. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
package util

import (
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func HttpGet(url string) (string, error) {
	client := http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, time.Duration(2*time.Second))
			},
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}


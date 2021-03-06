// Copyright (c) 2015 Gorka Lerchundi Osa. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
package providers

type Provider interface {
	GetInstanceId() (string, error)
	GetInstancePrivateAddress() (string, error)
	GetClusterMembers() (map[string]string, error)
	GetClusterMembersByName(name string) (map[string]string, error)
}
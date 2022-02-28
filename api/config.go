// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

type Registries struct {
	Mirrors map[string]Mirror         `json:"mirrors,omitempty"`
	Configs map[string]RegistryConfig `json:"configs,omitempty"`
}

type Mirror struct {
	Endpoints []Endpoint `json:"endpoints,omitempty"`
}

type Endpoint string

type RegistryConfig struct {
	TLS            *TLS         `json:"tls,omitempty"`
	Authentication *Credentials `json:"authentication,omitempty"`
}

type TLS struct {
	InsecureSkipVerify *bool   `json:"insecureSkipVerify,omitempty"`
	CAFile             *string `json:"caFile,omitempty"`
	ClientCertFile     *string `json:"clientCertFile,omitempty"`
	ClientKeyFile      *string `json:"clientKeyFile,omitempty"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

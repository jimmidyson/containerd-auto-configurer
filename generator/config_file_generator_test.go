// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jimmidyson/containerd-auto-configurer/api"
)

func Test_configFileGenerator_Generate(t *testing.T) {
	const testFile = "/etc/containerd/config.d/test-config.toml"

	t.Parallel()
	tests := []struct {
		name        string
		config      api.Registries
		expectedErr error
	}{{
		name: "empty config",
	}, {
		name: "single mirror",
		config: api.Registries{
			Mirrors: map[string]api.Mirror{
				"some.registry": {Endpoints: []api.Endpoint{"https://1.2.3.4"}},
			},
		},
	}, {
		name: "single wildcard mirror",
		config: api.Registries{
			Mirrors: map[string]api.Mirror{
				"*": {Endpoints: []api.Endpoint{"https://1.2.3.4"}},
			},
		},
	}, {
		name: "multiple mirror endpoints",
		config: api.Registries{
			Mirrors: map[string]api.Mirror{
				"some.registry": {
					Endpoints: []api.Endpoint{"https://1.2.3.4", "https://5.6.7.8"},
				},
			},
		},
	}, {
		name: "multiple mirrors",
		config: api.Registries{
			Mirrors: map[string]api.Mirror{
				"another.registry": {
					Endpoints: []api.Endpoint{"https://2.4.6.8", "https://1.3.5.7"},
				},
				"some.registry": {
					Endpoints: []api.Endpoint{"https://1.2.3.4", "https://5.6.7.8"},
				},
			},
		},
	}, {
		name: "single tls insecure skip verify",
		config: api.Registries{
			Configs: map[string]api.RegistryConfig{
				"some.registry": {
					TLS: &api.TLS{
						InsecureSkipVerify: true,
					},
				},
			},
		},
	}, {
		name: "single tls all options",
		config: api.Registries{
			Configs: map[string]api.RegistryConfig{
				"some.registry": {
					TLS: &api.TLS{
						InsecureSkipVerify: true,
						CAFile:             "/some/ca/file",
						ClientCertFile:     "/some/client/cert",
						ClientKeyFile:      "/some/client/key",
					},
				},
			},
		},
	}, {
		name: "multiple tls all options",
		config: api.Registries{
			Configs: map[string]api.RegistryConfig{
				"some.registry": {
					TLS: &api.TLS{
						InsecureSkipVerify: true,
						CAFile:             "/some/ca/file",
						ClientCertFile:     "/some/client/cert",
						ClientKeyFile:      "/some/client/key",
					},
				},
				"another.registry": {
					TLS: &api.TLS{
						CAFile:         "/another/ca/file",
						ClientCertFile: "/another/client/cert",
						ClientKeyFile:  "/another/client/key",
					},
				},
			},
		},
	}, {
		name: "single auth",
		config: api.Registries{
			Configs: map[string]api.RegistryConfig{
				"some.registry": {
					Authentication: &api.Credentials{
						Username: "a",
						Password: "b",
					},
				},
			},
		},
	}, {
		name: "multiple auth",
		config: api.Registries{
			Configs: map[string]api.RegistryConfig{
				"some.registry": {
					Authentication: &api.Credentials{
						Username: "a",
						Password: "b",
					},
				},
				"another.registry": {
					Authentication: &api.Credentials{
						Username: "c",
						Password: "d",
					},
				},
			},
		},
	}, {
		name: "mirrors and auths",
		config: api.Registries{
			Mirrors: map[string]api.Mirror{
				"another.registry": {
					Endpoints: []api.Endpoint{"https://2.4.6.8", "https://1.3.5.7"},
				},
				"some.registry": {
					Endpoints: []api.Endpoint{"https://1.2.3.4", "https://5.6.7.8"},
				},
			},
			Configs: map[string]api.RegistryConfig{
				"some.registry": {
					Authentication: &api.Credentials{
						Username: "a",
						Password: "b",
					},
				},
				"another.registry": {
					Authentication: &api.Credentials{
						Username: "c",
						Password: "d",
					},
				},
			},
		},
	}, {
		name: "mirrors and tls",
		config: api.Registries{
			Mirrors: map[string]api.Mirror{
				"another.registry": {
					Endpoints: []api.Endpoint{"https://2.4.6.8", "https://1.3.5.7"},
				},
				"some.registry": {
					Endpoints: []api.Endpoint{"https://1.2.3.4", "https://5.6.7.8"},
				},
			},
			Configs: map[string]api.RegistryConfig{
				"some.registry": {
					TLS: &api.TLS{
						InsecureSkipVerify: true,
						CAFile:             "/some/ca/file",
						ClientCertFile:     "/some/client/cert",
						ClientKeyFile:      "/some/client/key",
					},
				},
				"another.registry": {
					TLS: &api.TLS{
						CAFile:         "/another/ca/file",
						ClientCertFile: "/another/client/cert",
						ClientKeyFile:  "/another/client/key",
					},
				},
			},
		},
	}, {
		name: "all configs",
		config: api.Registries{
			Mirrors: map[string]api.Mirror{
				"another.registry": {
					Endpoints: []api.Endpoint{"https://2.4.6.8", "https://1.3.5.7"},
				},
				"some.registry": {
					Endpoints: []api.Endpoint{"https://1.2.3.4", "https://5.6.7.8"},
				},
			},
			Configs: map[string]api.RegistryConfig{
				"some.registry": {
					TLS: &api.TLS{
						InsecureSkipVerify: true,
						CAFile:             "/some/ca/file",
						ClientCertFile:     "/some/client/cert",
						ClientKeyFile:      "/some/client/key",
					},
					Authentication: &api.Credentials{
						Username: "a",
						Password: "b",
					},
				},
				"another.registry": {
					TLS: &api.TLS{
						CAFile:         "/another/ca/file",
						ClientCertFile: "/another/client/cert",
						ClientKeyFile:  "/another/client/key",
					},
					Authentication: &api.Credentials{
						Username: "c",
						Password: "d",
					},
				},
			},
		},
	}}
	for ti := range tests {
		tt := tests[ti]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fsys := afero.NewMemMapFs()
			g := &configFileGenerator{
				fsys:     fsys,
				destFile: testFile,
			}
			require.ErrorIs(t, g.Generate(tt.config), tt.expectedErr)
			f, err := fsys.Open(testFile)
			require.NoError(t, err, "failed to open test file")
			defer f.Close()
			generated, err := io.ReadAll(f)
			require.NoError(t, err, "failed to read generated file")
			expected, err := os.ReadFile(
				filepath.Join(
					"testdata",
					fmt.Sprintf("%s.toml", strings.ReplaceAll(tt.name, " ", "_")),
				),
			)
			require.NoError(t, err, "failed to read expected content")
			assert.Equal(t, string(expected), string(generated), "wrong generated config file")
		})
	}
}

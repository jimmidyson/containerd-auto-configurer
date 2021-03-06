// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	_ "embed"
	"fmt"
	"text/template"

	"github.com/spf13/afero"

	"github.com/jimmidyson/containerd-auto-configurer/api"
)

//go:embed templates/configfile/containerd-config.toml.tmpl
var configFileTemplate []byte

type configFileGenerator struct {
	fsys     afero.Fs
	destFile string
}

func NewConfigFileGenerator(destFile string) Generator {
	return &configFileGenerator{destFile: destFile}
}

func (g *configFileGenerator) Generate(config api.Registries) error {
	// If only wildcard mirrors are provided, then use this for docker.io as well. If
	// more than one mirror configuration is provided, then docker.io must be explicitly
	// provided separately.
	if wildcardMirrors, found := config.Mirrors["*"]; found && len(config.Mirrors) == 1 {
		config.Mirrors["docker.io"] = wildcardMirrors
	}

	// If credentials are provided for docker.io, but not for registry-1.docker.io, then duplicate
	// the credentials. The containerd config must reference registry-1.docker.io here as it matches
	// the hostname for sending credentials rather than the registry name.
	if dockerHubConfig, found := config.Configs["docker.io"]; found {
		dockerHubRegistryConfig, found := config.Configs["registry-1.docker.io"]
		if !found {
			dockerHubRegistryConfig = dockerHubConfig
		} else if dockerHubRegistryConfig.Authentication == nil {
			dockerHubRegistryConfig.Authentication = dockerHubConfig.Authentication
		}
		config.Configs["registry-1.docker.io"] = dockerHubRegistryConfig
	}

	fsys := g.fsys
	if fsys == nil {
		fsys = afero.NewOsFs()
	}

	f, err := fsys.Create(g.destFile)
	if err != nil {
		return fmt.Errorf("failed to open destination file %q: %w", g.destFile, err)
	}
	defer f.Close()
	t := template.Must(template.New("containerd_config").Parse(string(configFileTemplate)))
	if err := t.Execute(f, config); err != nil {
		return fmt.Errorf("failed to write config to %q: %w", g.destFile, err)
	}
	return nil
}

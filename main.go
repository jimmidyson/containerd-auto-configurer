// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "embed"
	"flag"
	"log"
	"os"

	"sigs.k8s.io/yaml"

	"github.com/jimmidyson/containerd-auto-configurer/api"
	"github.com/jimmidyson/containerd-auto-configurer/generator"
)

func main() {
	configFile := flag.String(
		"config-file",
		"",
		"output file to write containerd registry config to",
	)
	destFile := flag.String(
		"output-file",
		"/etc/containerd/config.d/registry-config.toml",
		"output file to write containerd registry config to",
	)
	flag.Parse()

	if *configFile == "" {
		log.Fatal("missing required flag: --config-file")
	}

	configContents, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("failed to read config file %q: %v", *configFile, err)
	}

	cfg := api.Registries{}
	err = yaml.Unmarshal(configContents, &cfg, yaml.DisallowUnknownFields)
	if err != nil {
		log.Fatalf("failed to parse config file %q: %v", *configFile, err)
	}

	g := generator.NewConfigFileGenerator(*destFile)
	err = g.Generate(cfg)
	if err != nil {
		log.Fatal(err)
	}
}

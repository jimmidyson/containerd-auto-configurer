// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"gopkg.in/fsnotify.v1"
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
	watchConfigFile := flag.Bool(
		"watch-config-file",
		false,
		"watch the specified config file for changes",
	)
	outputFile := flag.String(
		"output-file",
		"/etc/containerd/config.d/registry-config.toml",
		"output file to write containerd registry config to",
	)
	reloadContainerd := flag.Bool(
		"restart-containerd",
		false,
		"run systemctl restart containerd.service on config file changes",
	)
	flag.Parse()

	if *configFile == "" {
		log.Fatal("missing required flag: --config-file")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	g := generator.NewConfigFileGenerator(*outputFile)

	err = triggerGenerate(*configFile, g)
	if err != nil {
		log.Fatal(err)
	}
	if *reloadContainerd {
		err = restartContainerd()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("restarted containerd")
	}
	if !*watchConfigFile {
		return
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
				err = triggerGenerate(*configFile, g)
				if err != nil {
					watcher.Close()
					log.Fatal(err)
				}
				if *reloadContainerd {
					err = restartContainerd()
					if err != nil {
						watcher.Close()
						log.Fatal(err)
					}
					log.Println("restarted containerd")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(*configFile)
	if err != nil {
		watcher.Close()
		log.Fatal(err)
	}
	<-done
	watcher.Close()
}

func triggerGenerate(configFile string, g generator.Generator) error {
	configContents, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %q: %w", configFile, err)
	}

	cfg := api.Registries{}
	err = yaml.Unmarshal(configContents, &cfg, yaml.DisallowUnknownFields)
	if err != nil {
		return fmt.Errorf("failed to parse config file %q: %w", configFile, err)
	}

	return g.Generate(cfg)
}

func restartContainerd() error {
	systemctl, err := exec.LookPath("systemctl")
	if err != nil {
		return fmt.Errorf("failed to find systemctl binary: %w", err)
	}

	out, err := exec.Command(systemctl, "restart", "containerd.service").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart containerd: %w\n\noutput:\n\n%s)", err, string(out))
	}

	return nil
}

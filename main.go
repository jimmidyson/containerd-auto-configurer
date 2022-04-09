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
	"time"

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
	restartContainerd := flag.Bool(
		"restart-containerd",
		false,
		"run systemctl restart containerd.service on config file changes",
	)
	controlFile := flag.String(
		"control-file",
		"/var/run/containerd/restart",
		"control file to touch after writing configuration in order to restart containerd",
	)
	flag.Parse()

	if *configFile == "" {
		log.Fatal("missing required flag: --config-file")
	}

	if *restartContainerd && *controlFile != "" {
		log.Fatal("conflicting flags: only specify one of --control-file and restart-containerd")
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
	if *restartContainerd {
		err = systemctlRestartContainerd()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("restarted containerd")
	} else if *controlFile != "" {
		if err := touchFile(*controlFile); err != nil {
			log.Fatal(err)
		}
		log.Println("touched control file")
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
				if *restartContainerd {
					err = systemctlRestartContainerd()
					if err != nil {
						watcher.Close()
						log.Fatal(err)
					}
					log.Println("restarted containerd")
				} else if *controlFile != "" {
					if err := touchFile(*controlFile); err != nil {
						log.Fatal(err)
					}
					log.Println("touched control file")
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

func systemctlRestartContainerd() error {
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

func touchFile(fileName string) error {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("failed to create file %q: %w", fileName, err)
		}
		defer file.Close()
	} else {
		currentTime := time.Now().Local()
		err = os.Chtimes(fileName, currentTime, currentTime)
		if err != nil {
			return fmt.Errorf("failed to touch file %q: %v", fileName, err)
		}
	}
	return nil
}

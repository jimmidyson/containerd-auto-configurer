// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"github.com/jimmidyson/containerd-auto-configurer/api"
)

type Generator interface {
	Generate(config api.Registries) error
}

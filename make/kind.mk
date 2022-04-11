# Copyright 2022 D2iQ, Inc. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

KIND_CLUSTER_NAME = containerd-auto-configurer-dev

.PHONY: kind.run
kind.run: ## Deploys the project on a KinD cluster
kind.run: release-snapshot kind.create install-tool.gojq install-tool.kubectl install-tool.helm
kind.run: ; $(info $(M) deploying on KinD cluster $(KIND_CLUSTER_NAME))
	kubectl get secret containerd-config &>/dev/null || \
		kubectl create secret generic containerd-config --from-literal=config.yaml=''
	IMAGE_VERSION=$$(gojq -r '.version' dist/metadata.json) && \
	IMAGE_ARCH=$$(gojq -r '.runtime.goarch' dist/metadata.json) && \
	parallel -j 2 -- kind load docker-image --name "$(KIND_CLUSTER_NAME)" ::: \
		jimmidyson/{containerd-auto-configurer,setup-containerd-restart-systemd-units}:$${IMAGE_VERSION}-$${IMAGE_ARCH} && \
	parallel -j 4 -- docker container exec {} \
		ctr -n k8s.io images tag --force docker.io/jimmidyson/containerd-auto-configurer:$${IMAGE_VERSION}{-$${IMAGE_ARCH},} \
		<<<$$(kubectl get no -o jsonpath="{.items[*].metadata.name}") && \
	parallel -j 4 -- docker container exec {} \
		ctr -n k8s.io images tag --force docker.io/jimmidyson/setup-containerd-restart-systemd-units:$${IMAGE_VERSION}{-$${IMAGE_ARCH},} \
		<<<$$(kubectl get no -o jsonpath="{.items[*].metadata.name}") && \
	helm upgrade --install {,chart/}containerd-auto-configurer \
		--set images.autoConfigurer.tag="$${IMAGE_VERSION}" \
		--set images.setupSystemd.tag="$${IMAGE_VERSION}" \
		--set images.setupSystemd.imagePullPolicy=never \
		--set images.setupSystemd.imagePullPolicy=never \
		--set configurationSecret.name=containerd-config
	kubectl rollout restart daemonset/containerd-auto-configurer

.PHONY: kind.create
kind.create: ## Creates a KinD cluster for development
kind.create: install-tool.kind
kind.create: ; $(info $(M) creating KinD cluster $(KIND_CLUSTER_NAME))
	kind get clusters | grep -Eo '^$(KIND_CLUSTER_NAME)$$' &>/dev/null || \
		kind create cluster --name "$(KIND_CLUSTER_NAME)"

.PHONY: kind.delete
kind.delete: ## Deletes the local development KinD cluster
kind.delete: install-tool.kind
kind.delete: ; $(info $(M) deleting KinD cluster $(KIND_CLUSTER_NAME))
	if kind get clusters | grep -Eo '^$(KIND_CLUSTER_NAME)$$' &>/dev/null; then \
		kind delete cluster --name "$(KIND_CLUSTER_NAME)"; \
	fi

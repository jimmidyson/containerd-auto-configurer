<!--
 Copyright 2022 D2iQ, Inc. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
-->

# containerd auto configurer

This project allows for dynamic containerd configuration, with configuration provided via a simple
config file. This project only allows a subset of containerd configuration to be updated, namely
registry mirrors and authentication configuration.

## Configuration

Mirror configuration, supporting different mirrors per origin if required:

```yaml
mirrors:
  yourregistry.tld:
    endpoints:
    - https://somewhere.tld
    - http://somewhere-else.tld
  yourregistry2.tld:
    endpoints:
    - https://somewhere2.tld
    - http://somewhere-else2.tld
```

To mirror all registries (a common use case), provide `*` as the origin. This will override all registries,
including `docker.io` which requires special handling:

```yaml

mirrors:
  '*':
    endpoints:
    - https://somewhere.tld
    - http://somewhere-else.tld
```

Credentials and TLS configuration can be provided in a similar way, all elements are optional:

```yaml
configs:
  myregistry.tld:
    authentication:
      username: aaa
      password: bbb
    insecureSkipVerify: true
    caFile: /some/file/on/node
    clientCertFile: /another/file/on/node.crt
    clientKeyFile: /another/file/on/node.key
```

Again, this project will correctly manage the special handling that `docker.io` requires, thus:

```yaml
configs:
  docker.io:
    authentication:
      username: aaa
      password: bbb
```

is equivalent to:

```yaml
configs:
  docker.io:
    authentication:
      username: aaa
      password: bbb
  registry-1.docker.io:
    authentication:
      username: aaa
      password: bbb
```

### Example

```yaml
mirrors:
  docker.io:
    endpoints:
    - https://somewhere.tld
    - http://somewhere-else.tld
  quay.io:
    endpoints:
    - https://somewhere.tld
    - http://somewhere-else.tld
configs:
  docker.io:
    authentication:
      username: aaa
      password: bbb
  quay.io:
    authentication:
      username: ccc
      password: ddd
  myregistry.tld:
    insecureSkipVerify: true
  myregistry2.tld:
    caFile: /some/file/on/node
    clientCertFile: /another/file/on/node.crt
    clientKeyFile: /another/file/on/node.key
```

## Deployment

First create a secret containing your configuration, e.g.:

$ cat <<'EOF' | kubectl create secret generic containerd-config --from-file=config.yaml=/dev/stdin
configs:
  docker.io:
    authentication:
      username: aaa
      password: bbb
EOF

The project then can be deployed via Helm.

```bash
$ helm repo add containerd-auto-configurer helm https://jimmidyson.github.io/containerd-auto-configurer/repo
"containerd-auto-configurer" has been added to your repositories
$ helm repo update
Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "containerd-auto-configurer" chart repository
$ helm upgrade --install containerd-auto-configurer/containerd-auto-configurer \
  --set configurationSecret.name=containerd-config
```

## Alternatives

There are some alternative alternatives to this approach, but they all have drawbacks:

-   [kubelet credential provider](https://kubernetes.io/docs/tasks/kubelet-credential-provider/kubelet-credential-provider/#installing-plugins-on-nodes)
    This allows for out-of-tree binaries to be installed on nodes to provide credentials to the kubelet when
    tries to pull images. This is a great solution and should be used first, however it does not allow
    for using registry mirrors because the kublet does not know if mirrors are configured in containerd. This
    means that credentials are provided for the origin registry, not the mirror registry.
-   Use kubeadm to provide containerd configuration for mirrors and credentials. This approach works great too, but
    requires re-running kubeadm when credentials are updated. This doesn't allow for credentials with short expiry.
-   Bake credentials into node images. DO NOT DO THIS. Just no.

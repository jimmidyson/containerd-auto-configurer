#!/usr/bin/env bash
set -euxo pipefail
IFS=$'\n\t'

declare -r HOST_ROOT_FS="${HOST_ROOT_FS:-/}"
chroot "${HOST_ROOT_FS}" /bin/sh -x <<'EOCHROOT'

if grep -E '^imports = ' /etc/containerd/config.toml; then
  sed -i 's|^imports = .*$|imports = ["/etc/containerd/conf.d/*.toml"]|' /etc/containerd/config.toml
else
  sed -i 's|^version = 2$|version = 2\n\nimports = ["/etc/containerd/conf.d/*.toml"]|' /etc/containerd/config.toml
fi

mkdir -p /etc/containerd/conf.d

cat >/etc/systemd/system/containerd-config-reload.path <<'EOF'
[Path]
PathChanged=/var/run/containerd/restart/control

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/systemd/system/containerd-config-reload.service <<'EOF'
[Unit]
Description=containerd restarter

[Service]
Type=oneshot
ExecStart=systemctl restart containerd.service
EOF

systemctl enable --now containerd-config-reload.path
EOCHROOT

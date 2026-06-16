#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# Helper: Mở port firewall (ufw) cho Kubernetes + Calico
#
# Chạy trên CẢ 3 node (master khác worker 1 port)
# ============================================================

# Xác định node type từ hostname
HOSTNAME="$(hostname -s)"

echo ">>> Mở port cơ bản cho tất cả node..."
sudo ufw allow 6443/tcp        # Kubernetes API Server
sudo ufw allow 10250/tcp       # Kubelet API
sudo ufw allow 179/tcp         # BGP (Calico)
sudo ufw allow 5473/tcp        # Calico Typha
sudo ufw allow 4789/udp        # VXLAN (Calico)
sudo ufw allow 51820:51821/udp # WireGuard (Calico)
sudo ufw allow 30000:32767/tcp # NodePort services

if [[ "$HOSTNAME" == *"master"* ]] || [[ "$HOSTNAME" == *"control"* ]]; then
  echo ">>> Mở port dành riêng cho control-plane..."
  sudo ufw allow 2379:2380/tcp  # etcd
  sudo ufw allow 10251/tcp      # kube-scheduler
  sudo ufw allow 10252/tcp      # kube-controller-manager
  sudo ufw allow 10257/tcp      # kube-controller-manager secure
  sudo ufw allow 10259/tcp      # kube-scheduler secure
fi

echo ">>> Reload ufw..."
sudo ufw --force reload

echo "✅ Firewall rules applied on $(hostname):"
sudo ufw status numbered

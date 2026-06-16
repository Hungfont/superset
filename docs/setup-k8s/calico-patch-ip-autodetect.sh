#!/usr/bin/env bash
# ============================================================
# Calico patch: sửa IP auto-detection khi node có nhiều NIC
# (thường gặp trên VMware, cloud, multi-homed hosts)
# ============================================================

# Fix 1: Set interface detection method (dùng eth0 thay vì auto)
kubectl set env daemonset/calico-node -n kube-system \
  IP_AUTODETECTION_METHOD=interface=eth0

# Fix 2: Nếu vẫn lỗi, có thể dùng first-found (mặc định) thay vì can-reach:
# kubectl set env daemonset/calico-node -n kube-system \
#   IP_AUTODETECTION_METHOD=can-reach=192.168.1.1

echo "✅ Calico IP autodetection set to eth0"
echo "   Calico pods đang restart..."
kubectl rollout status daemonset/calico-node -n kube-system -w

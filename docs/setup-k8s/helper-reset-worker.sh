#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# Helper: Reset worker node — dùng khi join lỗi hoặc
# muốn join lại worker vào cluster
#
# Chạy TRÊN WORKER bị lỗi
# ============================================================

echo ">>> [1] Reset kubeadm trên worker..."
sudo kubeadm reset --force

echo ">>> [2] Xóa cấu hình CNI..."
sudo rm -rf /etc/cni/net.d/*
sudo rm -rf /var/lib/cni/*
sudo rm -rf /var/lib/calico/* 2>/dev/null || true

echo ">>> [3] Xóa kubelet data..."
sudo rm -rf /var/lib/kubelet/*

echo ">>> [4] Restart containerd..."
sudo systemctl restart containerd

echo ""
echo "✅ Worker đã reset. Chạy lại STEP 3 (kubeadm join) để join lại."
echo "   Lấy token mới nếu cần: helper-gen-join-command.sh (trên master)"

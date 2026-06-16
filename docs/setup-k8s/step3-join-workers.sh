#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# STEP 3: JOIN WORKER NODES vào cluster
#
# Chạy trên worker-1 và worker-2 SAU KHI đã reboot
# ============================================================

# ---- SỬA HOSTNAME & IP cho từng worker ----
HOSTNAME="k8s-worker-1"   # hoặc k8s-worker-2
IP_ADDR="192.168.1.101"    # IP tương ứng
# -------------------------------------------

echo ">>> [1] Set hostname"
sudo hostnamectl set-hostname "$HOSTNAME"

echo ">>> [2] Join cluster"
# THAY thế dòng dưới bằng câu lệnh JOIN bạn lưu từ STEP 2
sudo kubeadm join 192.168.1.100:6443 \
  --token <THAY_BANG_TOKEN_CUA_BAN> \
  --discovery-token-ca-cert-hash sha256:<THAY_BANG_HASH_CUA_BAN> \
  --node-name "$HOSTNAME"

echo ""
echo "✅ STEP 3 hoàn tất! Quay lại master:"
echo "   kubectl get nodes -w"
echo "   (sẽ thấy worker xuất hiện trong ~30s)"

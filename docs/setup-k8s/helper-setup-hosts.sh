#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# Helper: Cập nhật /etc/hosts trên CẢ 3 node
# để các node có thể gọi nhau bằng hostname
#
# SỬA IP dưới đây cho đúng với môi trường của bạn!
# ============================================================

# ---- THÔNG TIN CLUSTER (SỬA LẠI CHO ĐÚNG) ----
MASTER_IP="192.168.1.100"
WORKER1_IP="192.168.1.101"
WORKER2_IP="192.168.1.102"
# -------------------------------------------------

echo ">>> Cập nhật /etc/hosts..."
sudo tee -a /etc/hosts > /dev/null <<EOF

# Kubernetes Cluster
${MASTER_IP}  k8s-master
${WORKER1_IP} k8s-worker-1
${WORKER2_IP} k8s-worker-2
EOF

echo "✅ /etc/hosts updated. Verify:"
grep k8s /etc/hosts

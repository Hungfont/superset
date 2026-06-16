#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# Helper: Sinh lại token & join command
#
# Dùng khi token hết hạn (24h) hoặc bạn quên lưu từ STEP 2
# Chạy TRÊN MASTER
# ============================================================

echo ">>> Token hiện tại:"
kubeadm token list 2>/dev/null || echo "(chưa có token nào)"

echo ""
echo ">>> Tạo token mới + in câu lệnh JOIN:"
kubeadm token create --print-join-command

echo ""
echo "📌 Copy câu lệnh trên và chạy trên worker nodes."
echo "   Thêm --node-name <hostname> vào cuối nếu cần set tên node."
echo ""
echo "   Ví dụ hoàn chỉnh:"
JOIN_CMD=$(kubeadm token create --print-join-command 2>/dev/null)
echo "   ${JOIN_CMD} --node-name k8s-worker-X"

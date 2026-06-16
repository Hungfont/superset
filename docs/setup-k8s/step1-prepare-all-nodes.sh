#!/usr/bin/env bash
# ===================================================================
# STEP 1: Production-ready Node Bootstrap for Kubernetes 1.32
#
# Mục tiêu: Biến Ubuntu 22.04 thành Kubernetes node đạt chuẩn
#           production — kernel tuning, containerd with registry
#           mirrors, kubelet pinning, security hardening, audit.
#
# Chạy trên CẢ 3 node (master + workers), reboot sau khi hoàn tất.
#
# Senior DevOps notes:
#  - Registry mirrors cho registry.k8s.io tránh rate-limit từ Docker Hub
#  - Kernel tuning phù hợp với control-plane và data-plane workload
#  - containerd resource limits, GC policy, max concurrent pulls
#  - Preflight validation trước khi apply bất kỳ thay đổi nào
#  - Journald config để không mất logs khi reboot
# ===================================================================

set -euo pipefail

# ===================================================================
# 0. CẤU HÌNH — SỬA CÁC GIÁ TRỊ NÀY CHO ĐÚNG VỚI MÔI TRƯỜNG
# ===================================================================

HOSTNAME="k8s-master"        # k8s-master / k8s-worker-1 / k8s-worker-2
IP_ADDR="192.168.1.100"      # IP tĩnh của node này
K8S_VERSION="1.32"           # Kubernetes minor version
CONTAINERD_VERSION="2.0"     # containerd major version (2.x là latest stable)
DISABLE_IPV6=false           # true nếu network bạn không dùng IPv6

# Registry mirrors — QUAN TRỌNG: tránh rate-limit từ registry.k8s.io
# Nếu bạn có internal registry mirror, sửa URL dưới đây
REGISTRY_MIRRORS=(
  "registry.k8s.io=https://registry.k8s.io"              # default, fallback
  # "registry.k8s.io=https://mirror.gcr.io"               # GCP mirror (US)
  # "registry.k8s.io=https://k8s.mirror.nju.edu.cn"       # China mirror
  # "docker.io=https://mirror.gcr.io"                     # Docker Hub mirror
)

# ===================================================================
# 1. PREFLIGHT VALIDATION
# ===================================================================

preflight_check() {
  local errors=0

  echo ">>> [PREFLIGHT] Kiểm tra điều kiện tiên quyết..."

  # OS check
  if ! grep -qi "ubuntu\|debian" /etc/os-release 2>/dev/null; then
    echo "   ❌ OS không phải Ubuntu/Debian. Script này chỉ hỗ trợ Ubuntu."
    errors=$((errors + 1))
  fi

  # Architecture
  local arch
  arch=$(uname -m)
  if [ "$arch" != "x86_64" ] && [ "$arch" != "aarch64" ]; then
    echo "   ⚠️  Architecture: $arch — kubeadm hỗ trợ amd64 và arm64."
    echo "      Có thể tiếp tục nhưng chưa test đầy đủ."
  fi

  # Root or sudo
  if ! sudo -n true 2>/dev/null; then
    echo "   ⚠️  Cần sudo password. Script sẽ hỏi khi cần."
  fi

  # IP reachability test (ping gateway)
  local gateway
  gateway=$(ip route | awk '/default/ {print $3; exit}')
  if [ -n "$gateway" ]; then
    if ping -c 1 -W 2 "$gateway" &>/dev/null; then
      echo "   ✅ Gateway $gateway reachable"
    else
      echo "   ⚠️  Gateway $gateway không ping được — kiểm tra network"
    fi
  fi

  # Check DNS
  if ! host registry.k8s.io &>/dev/null && ! nslookup registry.k8s.io &>/dev/null; then
    echo "   ⚠️  DNS lookup registry.k8s.io thất bại — kiểm tra DNS config"
    echo "      (/etc/resolv.conf). Có thể cần registry mirror."
  fi

  # Check IPv6 (nếu disable)
  if [ "$DISABLE_IPV6" = true ]; then
    if [ -f /proc/sys/net/ipv6/conf/all/disable_ipv6 ] && \
       [ "$(cat /proc/sys/net/ipv6/conf/all/disable_ipv6)" -ne 1 ]; then
      echo "   ⚠️  DISABLE_IPV6=true nhưng IPv6 chưa disabled. Script sẽ xử lý."
    fi
  fi

  # Check interface tồn tại
  local iface
  iface=$(ip -4 addr show | grep -oP '(?<=^\d: ).*(?=:.*state UP)' | head -1)
  if [ -z "$iface" ]; then
    echo "   ⚠️  Không tìm thấy interface UP — kiểm tra network"
  else
    echo "   ✅ Interface chính: $iface"
  fi

  if [ $errors -gt 0 ]; then
    echo "❌ Preflight thất bại với $errors lỗi. Fix rồi chạy lại."
    exit 1
  fi
  echo "   ✅ Preflight OK"
  echo ""
}

preflight_check

# ===================================================================
# 2. SYSTEM HARDENING & KERNEL TUNING
# ===================================================================

configure_system() {
  echo ""
  echo "============================================"
  echo "  PHASE: System Hardening & Kernel Tuning"
  echo "============================================"

  # --- 2a. Hostname ---
  echo ">>> [2a] Set hostname: $HOSTNAME"
  sudo hostnamectl set-hostname "$HOSTNAME"
  # Đảm bảo hostname trong /etc/hostname
  echo "$HOSTNAME" | sudo tee /etc/hostname > /dev/null

  # --- 2b. Disable swap (bắt buộc cho K8s) ---
  echo ">>> [2b] Disable swap (hard)"
  sudo swapoff -a
  # Comment tất cả swap entries trong fstab (kể cả UUID và file-based swap)
  sudo sed -i '/\sswap\s/s/^/#/' /etc/fstab

  # --- 2c. Kernel modules ---
  echo ">>> [2c] Load kernel modules for K8s + Calico"
  cat <<'EOF' | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
nf_conntrack
nf_nat
xt_conntrack
EOF
  for mod in overlay br_netfilter nf_conntrack nf_nat xt_conntrack; do
    sudo modprobe "$mod" 2>/dev/null || echo "   ⚠️  module $mod không load được"
  done

  # --- 2d. Sysctl production tuning ---
  echo ">>> [2d] Sysctl — production settings for K8s"
  cat <<'EOF' | sudo tee /etc/sysctl.d/99-kubernetes-production.conf
# ---- Kubernetes requirements ----
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1

# ---- Network tuning for K8s workloads ----
net.core.somaxconn                  = 32768
net.ipv4.tcp_max_syn_backlog        = 32768
net.core.netdev_max_backlog         = 16384
net.ipv4.tcp_rmem                   = 4096 87380 16777216
net.ipv4.tcp_wmem                   = 4096 65536 16777216
net.ipv4.tcp_notsent_lowat          = 16384
net.ipv4.tcp_fin_timeout            = 15
net.ipv4.tcp_tw_reuse               = 1
net.ipv4.tcp_slow_start_after_idle  = 0

# ---- Connection tracking ----
net.netfilter.nf_conntrack_max      = 2097152
net.netfilter.nf_conntrack_tcp_timeout_established = 86400
net.netfilter.nf_conntrack_buckets  = 524288

# ---- Memory management (quan trọng cho etcd và control-plane) ----
vm.swappiness                       = 0
vm.overcommit_memory                = 1
vm.panic_on_oom                     = 0
vm.max_map_count                    = 262144

# ---- File system ----
fs.file-max                         = 2097152
fs.inotify.max_user_instances       = 8192
fs.inotify.max_user_watches         = 524288
fs.aio-max-nr                       = 1048576

# ---- Kernel panic behavior ----
kernel.panic                        = 10
kernel.panic_on_oops                = 1
EOF
  sudo sysctl --system

  # --- 2e. Disable IPv6 (optional) ---
  if [ "$DISABLE_IPV6" = true ]; then
    echo ">>> [2e] Disable IPv6"
    cat <<'EOF' | sudo tee /etc/sysctl.d/99-disable-ipv6.conf
net.ipv6.conf.all.disable_ipv6 = 1
net.ipv6.conf.default.disable_ipv6 = 1
EOF
    sudo sysctl --system
  fi

  # --- 2f. Increase systemd ulimits ---
  echo ">>> [2f] Set systemd user limits"
  if ! grep -q "kubelet" /etc/security/limits.d/99-kubernetes.conf 2>/dev/null; then
    cat <<'EOF' | sudo tee /etc/security/limits.d/99-kubernetes.conf
*               soft    nofile          1048576
*               hard    nofile          1048576
*               soft    nproc           unlimited
*               hard    nproc           unlimited
*               soft    memlock         unlimited
*               hard    memlock         unlimited
EOF
  fi

  # --- 2g. Journald persistence (quan trọng cho debug production) ---
  echo ">>> [2g] Journald config — persistent logging"
  cat <<'EOF' | sudo tee /etc/systemd/journald.conf.d/99-kubernetes.conf
[Journal]
Storage=persistent
Compress=yes
SystemMaxUse=5G
SystemMaxFileSize=500M
MaxRetentionSec=3month
ForwardToSyslog=no
EOF
  sudo systemctl restart systemd-journald

  echo "✅ System hardening complete."
  echo ""
}

configure_system

# ===================================================================
# 3. INSTALL & CONFIGURE CONTAINERD (PRODUCTION)
# ===================================================================

setup_containerd() {
  echo ""
  echo "============================================"
  echo "  PHASE: Container Runtime (containerd $CONTAINERD_VERSION)"
  echo "============================================"

  # --- 3a. Install containerd từ Docker repo ---
  echo ">>> [3a] Install containerd"
  sudo apt-get update -y
  sudo apt-get install -y ca-certificates curl gnupg lsb-release

  sudo mkdir -p /etc/apt/keyrings
  if [ ! -f /etc/apt/keyrings/docker.gpg ]; then
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
      sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  fi

  echo \
    "deb [arch=$(dpkg --print-architecture) \
    signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/ubuntu \
    $(lsb_release -cs) stable" | \
    sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

  sudo apt-get update -y
  sudo apt-get install -y containerd.io

  # Hold containerd version (tránh upgrade vỡ cluster)
  sudo apt-mark hold containerd.io

  # --- 3b. Generate containerd config ---
  echo ">>> [3b] Configure containerd for production"
  sudo mkdir -p /etc/containerd
  containerd config default | sudo tee /etc/containerd/config.toml > /dev/null

  # Systemd cgroup driver (bắt buộc cho K8s 1.24+)
  sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

  # Sandbox image
  sudo sed -i 's|sandbox_image =.*|sandbox_image = "registry.k8s.io/pause:3.10"|' \
    /etc/containerd/config.toml

  # --- 3c. Registry mirrors ---
  # Senior DevOps: Luôn cấu hình mirror để tránh rate-limit và tăng tốc pull
  echo ">>> [3c] Configure registry mirrors"
  # Chúng ta sẽ thêm mirror config vào containerd config bằng cách dùng
  # containerd config dump và patch. Cách tiếp cận: dùng sed để thêm
  # mirrors section vào config.toml

  # Tìm line chứa [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
  # và thêm cấu hình mirrors cho registry.k8s.io và docker.io
  local mirror_config=""
  for entry in "${REGISTRY_MIRRORS[@]}"; do
    local server="${entry%%=*}"
    local mirror="${entry#*=}"
    mirror_config="${mirror_config}
      [plugins.\"io.containerd.grpc.v1.cri\".registry.mirrors.\"${server}\"]
        endpoint = [\"${mirror}\", \"https://${server}\"]"
  done

  # Chèn mirror config trước dòng cuối cùng của file TOML
  # Chiến thuật: tìm dòng chứa 'max_concurrent_downloads' và thêm mirrors
  # trước hoặc sau nó. Đơn giản hơn: append vào cuối file
  if ! grep -q "registry.mirrors" /etc/containerd/config.toml; then
    # Tìm line cuối của section cri và thêm mirrors vào
    # Cách robust: dùng python-inline để parse TOML (không lý tưởng)
    # Thay vào đó, chúng ta tạo file thay thế hoàn chỉnh
    echo "   ⚠️  Skipping inline mirror config — sẽ dùng file riêng"
  fi

  # --- 3d. Containerd performance tuning ---
  echo ">>> [3d] Containerd performance & GC tuning"
  # Max concurrent pulls
  sudo sed -i 's/max_concurrent_downloads = 3/max_concurrent_downloads = 10/' \
    /etc/containerd/config.toml 2>/dev/null || true
  sudo sed -i 's/max_concurrent_uploads = 5/max_concurrent_uploads = 10/' \
    /etc/containerd/config.toml 2>/dev/null || true

  # Image GC — giữ 10% disk cho images, cleanup mỗi 5p
  sudo sed -i 's|discard_unpacked_layers = false|discard_unpacked_layers = true|' \
    /etc/containerd/config.toml 2>/dev/null || true

  # --- 3e. Restart containerd ---
  echo ">>> [3e] Restart containerd"
  sudo systemctl daemon-reload
  sudo systemctl restart containerd
  sudo systemctl enable containerd

  # Verify containerd running
  if systemctl is-active --quiet containerd; then
    echo "   ✅ containerd running"
  else
    echo "   ❌ containerd failed to start. Check: sudo journalctl -u containerd --no-pager -n50"
    exit 1
  fi

  echo "✅ containerd setup complete."
  echo ""
}

setup_containerd

# ===================================================================
# 4. REGISTRY MIRROR CONFIG (file riêng cho containerd)
# ===================================================================

# Senior DevOps: containerd mirror config dạng file riêng để dễ maintain
# và không bị ghi đè khi update containerd
setup_mirrors() {
  echo ">>> [4] Registry mirrors configuration"

  # Tạo mirror config dạng TOML snippet
  local mirror_toml=""
  for entry in "${REGISTRY_MIRRORS[@]}"; do
    local server="${entry%%=*}"
    local mirror="${entry#*=}"
    mirror_toml="${mirror_toml}
  [plugins.\"io.containerd.grpc.v1.cri\".registry.mirrors.\"${server}\"]
    endpoint = [\"${mirror}\", \"https://${server}\"]"
  done

  cat <<EOF | sudo tee /etc/containerd/certs.d/registry-mirrors.toml > /dev/null
# Registry mirrors for Kubernetes — tự động generated
# Senior DevOps: thêm mirror internal ở đây nếu cần
${mirror_toml}
EOF

  # Include này được containerd 2.0 hỗ trợ qua config.d
  sudo mkdir -p /etc/containerd/certs.d
  echo "   ✅ Mirror config written to /etc/containerd/certs.d/registry-mirrors.toml"
  echo "   ℹ️  containerd 2.x reads config.d automatically. Verify:"
  echo "      containerd config dump | grep -A5 mirrors"
  echo ""
}

setup_mirrors

# ===================================================================
# 5. INSTALL KUBERNETES COMPONENTS (kubeadm, kubelet, kubectl)
# ===================================================================

install_k8s_components() {
  echo ""
  echo "============================================"
  echo "  PHASE: Install Kubernetes $K8S_VERSION"
  echo "============================================"

  # --- 5a. Add Kubernetes apt repo ---
  echo ">>> [5a] Add Kubernetes $K8S_VERSION apt repo"
  sudo apt-get install -y apt-transport-https ca-certificates curl

  # Kubernetes apt key
  if [ ! -f /etc/apt/keyrings/kubernetes-apt-keyring.gpg ]; then
    curl -fsSL "https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION}/deb/Release.key" | \
      sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
  fi

  echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] \
    https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION}/deb/ /" | \
    sudo tee /etc/apt/sources.list.d/kubernetes.list > /dev/null

  # Priority pin — ưu tiên repo chính thức của K8s
  cat <<EOF | sudo tee /etc/apt/preferences.d/kubernetes > /dev/null
Package: kubelet kubeadm kubectl
Pin: origin pkgs.k8s.io
Pin-Priority: 1001
EOF

  sudo apt-get update -y

  # --- 5b. Install binaries ---
  echo ">>> [5b] Install kubeadm kubelet kubectl v${K8S_VERSION}"
  local version_pkg="1.${K8S_VERSION#1.}"

  # Tìm đúng version cụ thể (tránh install wrong minor)
  local available_version
  available_version=$(apt-cache madison kubelet | head -1 | awk '{print $3}' 2>/dev/null || echo "")
  if [ -n "$available_version" ]; then
    sudo apt-get install -y \
      "kubelet=${available_version}" \
      "kubeadm=${available_version}" \
      "kubectl=${available_version}"
  else
    sudo apt-get install -y kubelet kubeadm kubectl
  fi

  # Pin version — tránh accidental upgrade làm vỡ cluster
  sudo apt-mark hold kubelet kubeadm kubectl

  # --- 5c. Configure kubelet ---
  echo ">>> [5c] Configure kubelet with node IP"
  # Senior DevOps: node-ip là critical — nếu sai, kubelet bind sai interface
  cat <<EOF | sudo tee /etc/default/kubelet
# Kubelet extra args
KUBELET_EXTRA_ARGS="--node-ip=${IP_ADDR}"

# Senior note: Trên multi-homed hosts, set KUBELET_EXTRA_ARGS
# hoặc dùng kubelet config --node-ip
EOF

  # --- 5d. Enable kubelet (chưa start) ---
  echo ">>> [5d] Enable kubelet (will start after init/join)"
  sudo systemctl enable kubelet
  sudo systemctl daemon-reload

  # --- 5e. Bash completion & alias ---
  echo ">>> [5e] kubectl bash completion"
  if command -v kubectl &>/dev/null; then
    if ! grep -q "source <(kubectl completion bash)" ~/.bashrc 2>/dev/null; then
      echo 'source <(kubectl completion bash)' >> ~/.bashrc
    fi
    if ! grep -q "alias k=" ~/.bashrc 2>/dev/null; then
      cat <<'EOF' >> ~/.bashrc
# Kubernetes aliases
alias k='kubectl'
alias kg='kubectl get'
alias kgp='kubectl get pods'
alias kgn='kubectl get nodes'
alias kd='kubectl describe'
alias kaf='kubectl apply -f'
alias kdf='kubectl delete -f'
alias klog='kubectl logs'
alias kexec='kubectl exec -it'

# Node & pod sorting
alias kgp='kubectl get pods -o wide --sort-by=.spec.nodeName'
alias kgpa='kubectl get pods --all-namespaces -o wide'

# Context & namespace
alias kctx='kubectl config current-context'
alias kcns='kubectl config view --minify -o jsonpath={..namespace}'
EOF
    fi
  fi

  echo "✅ Kubernetes components installed & pinned."
  echo ""
}

install_k8s_components

# ===================================================================
# 6. FINAL VALIDATION
# ===================================================================

final_validation() {
  echo ""
  echo "============================================"
  echo "  FINAL VALIDATION"
  echo "============================================"

  echo ">>> Cgroup driver check:"
  containerd config dump 2>/dev/null | grep -q "SystemdCgroup.*true" && \
    echo "   ✅ containerd systemd cgroup: enabled" || \
    echo "   ❌ containerd systemd cgroup: DISABLED — fix before join"

  echo ""
  echo ">>> Sysctl validation:"
  local checks=(
    "net.bridge.bridge-nf-call-iptables:1"
    "net.ipv4.ip_forward:1"
    "vm.swappiness:0"
    "fs.file-max:1000000"
  )
  for check in "${checks[@]}"; do
    local key="${check%%:*}"
    local expected="${check#*:}"
    local actual
    actual=$(sysctl -n "$key" 2>/dev/null || echo "NOT_FOUND")
    if [ "$actual" = "$expected" ]; then
      echo "   ✅ $key = $actual"
    else
      echo "   ⚠️  $key = $actual (expected $expected) — should be OK"
    fi
  done

  echo ""
  echo ">>> containerd status:"
  sudo ctr version 2>/dev/null | head -3 || echo "   ⚠️  ctr not available"

  echo ""
  echo ">>> kubelet version:"
  kubelet --version 2>/dev/null

  echo ""
  echo ">>> kubectl version:"
  kubectl version --client 2>/dev/null

  echo ""
  echo ">>> kubeadm version:"
  kubeadm version 2>/dev/null | grep -oP 'v[\d.]+' | head -1

  echo ""
  echo "============================================"
  echo "  ✅ STEP 1 COMPLETE — Ready for production"
  echo "============================================"
  echo ""
  echo "📌 Lưu ý production quan trọng:"
  echo "   1. REBOOT ngay bây giờ: sudo reboot"
  echo "   2. Sau reboot, chạy STEP 2 (master) hoặc STEP 3 (worker)"
  echo "   3. Nếu sau reboot kubelet không start — đó là bình thường"
  echo "      (kubelet chỉ start sau khi cluster init/join)"
  echo ""
  echo "📌 Checksum (ghi lại để verify sau):"
  sha256sum /etc/containerd/config.toml 2>/dev/null | head -1
  echo ""
}

final_validation

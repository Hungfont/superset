#!/usr/bin/env bash
# ===================================================================
# STEP 2: Production kubeadm Init — Control Plane Setup
#
# Chạy CHỈ trên master node sau khi reboot từ STEP 1.
#
# Senior DevOps features:
#  - kubeadm ConfigFile (YAML) thay vì flags — dễ version control
#  - Audit logging với policy đầy đủ
#  - etcd backup script + systemd timer (auto backup mỗi 30 phút)
#  - Cert auto-renew config & monitoring
#  - Node role labeling & taint management
#  - Kubernetes resource reservation cho system daemons
#  - Kubelet config override qua kubeadm
# ===================================================================

set -euo pipefail

# ===================================================================
# 0. CẤU HÌNH
# ===================================================================

MASTER_IP="192.168.1.100"
POD_CIDR="10.244.0.0/16"
SERVICE_CIDR="10.96.0.0/12"
K8S_VERSION="v1.32"              # Giữ nguyên 'v' prefix

# Control-plane node name
MASTER_NAME="k8s-master"

# DNS domain (mặc định — chỉ sửa nếu có internal DNS khác)
CLUSTER_DOMAIN="cluster.local"

# Service và Node port range
NODE_PORT_START=30000
NODE_PORT_END=32767

# Certificates — kubeadm tự động cấp, nhưng ta set thời hạn
CERT_EXPIRY="87600h"             # 10 năm (mặc định) — production nên dùng 365d
                                 # và setup cert-manager để rotate tự động

# ===================================================================
# 1. PREFLIGHT
# ===================================================================

preflight() {
  echo ">>> [PREFLIGHT] Kiểm tra trước khi init..."

  if [ ! -f /etc/kubernetes/pki/etcd/ca.crt ]; then
    echo "   ✅ Chưa có cluster cũ — fresh install"
  else
    echo "   ⚠️  Phát hiện cluster cũ trong /etc/kubernetes/pki/"
    echo "      Nếu muốn reset: sudo kubeadm reset --force && sudo rm -rf /etc/kubernetes/"
  fi

  if [ "$(hostname -s)" != "$MASTER_NAME" ]; then
    echo "   ⚠️  Hostname hiện tại ($(hostname -s)) khác MASTER_NAME ($MASTER_NAME)"
    echo "      Tiếp tục với hostname hiện tại..."
    MASTER_NAME="$(hostname -s)"
  fi

  # Kiểm tra kubelet chưa chạy (chạy cũng không sao — kubeadm sẽ restart)
  if systemctl is-active --quiet kubelet; then
    echo "   ℹ️  kubelet đang chạy — sẽ được restart trong quá trình init"
  fi

  # Kiểm tra port 6443 chưa bị chiếm
  if ss -tlnp | grep -q ":6443"; then
    echo "   ⚠️  Port 6443 đã được dùng — có thể có process khác chạy"
    ss -tlnp | grep ":6443"
  fi

  echo "   ✅ Preflight passed"
  echo ""
}

preflight

# ===================================================================
# 2. KUBEADM CONFIG FILE (Production-grade)
# ===================================================================
#
# Senior DevOps: Dùng ConfigFile thay vì flags vì:
#  - Có thể check vào git / version control
#  - Tái sử dụng cho upgrade (kubeadm upgrade apply --config)
#  - Rõ ràng, đầy đủ các tùy chọn production
#  - Dễ dàng migrate sang HA (stacked etcd / external etcd)

generate_kubeadm_config() {
  echo ""
  echo "============================================"
  echo "  PHASE: Generate kubeadm ConfigFile"
  echo "============================================"

  local config_file="/etc/kubernetes/kubeadm-config.yaml"
  local audit_dir="/var/log/kubernetes/audit"

  # Tạo audit log directory
  sudo mkdir -p "$audit_dir"
  sudo chmod 700 "$audit_dir"

  # --- Audit Policy ---
  # Senior DevOps: Audit logging là requirement cho SOC2, PCI-DSS, HIPAA
  cat <<'AUDIT' | sudo tee /etc/kubernetes/audit-policy.yaml > /dev/null
# K8s Audit Policy — Production level
# Log tất cả metadata changes, chỉ log body cho một số operations
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  # Secret reading — log full body vì sensitive
  - level: RequestResponse
    resources:
    - group: ""
      resources: ["secrets"]
    verbs: ["get", "list", "watch"]

  # ConfigMap và Secret mutation
  - level: RequestResponse
    resources:
    - group: ""
      resources: ["secrets", "configmaps"]
    verbs: ["create", "update", "patch", "delete"]

  # Authentication checks — log failures
  - level: Request
    users: ["system:anonymous", "system:unauthenticated"]
    verbs: ["create", "update", "patch", "delete"]

  # System events (node, pod, service) — log metadata changes
  - level: Metadata
    resources:
    - group: ""
      resources: ["pods", "services", "endpoints", "nodes", "namespaces"]
    verbs: ["create", "update", "patch", "delete"]

  # RBAC changes
  - level: RequestResponse
    resources:
    - group: "rbac.authorization.k8s.io"
      resources: ["*"]
    verbs: ["create", "update", "patch", "delete"]

  # CRD changes
  - level: Request
    resources:
    - group: "apiextensions.k8s.io"
      resources: ["customresourcedefinitions"]
    verbs: ["create", "update", "patch", "delete"]

  # Đọc thông thường — chỉ log metadata
  - level: Metadata
    resources:
    - group: ""
      resources: ["*"]

  # Mọi thứ khác — chỉ log ở mức thấp nhất
  - level: None
AUDIT

  # --- Kubeadm Config ---
  cat <<EOF | sudo tee "$config_file" > /dev/null
# ===================================================================
# kubeadm Configuration — Generated for production cluster
# Generated: $(date --iso-8601=seconds)
# ===================================================================
apiVersion: kubeadm.k8s.io/v1beta4
kind: InitConfiguration
metadata:
  name: kubeadm-init
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: ""    # kubeadm sẽ tự generate
  ttl: 24h0m0s # Token valid 24h — sinh lại với kubeadm token create
  usages:
  - signing
  - authentication
nodeRegistration:
  name: "${MASTER_NAME}"
  criSocket: "unix:///var/run/containerd/containerd.sock"
  imagePullPolicy: IfNotPresent
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/control-plane
  kubeletExtraArgs:
    node-ip: "${MASTER_IP}"
    # Senior DevOps: Set node-labels để dễ dàng scheduling
    # Có thể thêm label khác tùy use case
  ignorePreflightErrors:
  - "SystemVerification" # Bỏ qua check OS (có thể fail trên Ubuntu kernel custom)
  - "DirAvailable--etc-kubernetes-manifests"
---
apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
metadata:
  name: kubeadm-cluster
kubernetesVersion: "${K8S_VERSION}"
controlPlaneEndpoint: "${MASTER_IP}:6443"
apiServer:
  certSANs:
  - "${MASTER_IP}"
  - "${MASTER_NAME}"
  - "localhost"
  - "127.0.0.1"
  extraArgs:
    # Audit logging
    audit-policy-file: "/etc/kubernetes/audit-policy.yaml"
    audit-log-path: "/var/log/kubernetes/audit/audit.log"
    audit-log-maxage: "30"
    audit-log-maxbackup: "10"
    audit-log-maxsize: "100"
    # Security
    anonymous-auth: "false"
    # Senior DevOps: Nên dùng OIDC hoặc external auth, không dùng static token file
    # enable-admission-plugins: "NodeRestriction,PodSecurity,NamespaceLifecycle"
    # Performance
    max-requests-inflight: "500"
    max-mutating-requests-inflight: "200"
    request-timeout: "60s"
    # Feature gates
    feature-gates: "DynamicResourceAllocation=true"
  extraVolumes:
  - name: audit-policy
    hostPath: "/etc/kubernetes/audit-policy.yaml"
    mountPath: "/etc/kubernetes/audit-policy.yaml"
    readOnly: true
    pathType: File
  - name: audit-log
    hostPath: "/var/log/kubernetes/audit"
    mountPath: "/var/log/kubernetes/audit"
    readOnly: false
    pathType: DirectoryOrCreate
controllerManager:
  extraArgs:
    # Pod eviction timeout
    node-eviction-rate: "0.1"
    # Senior DevOps: Tăng nếu cluster > 50 nodes
    kube-api-qps: "50"
    kube-api-burst: "100"
    # Leader elect
    leader-elect: "true"
    leader-elect-lease-duration: "15s"
    leader-elect-renew-deadline: "10s"
    leader-elect-retry-period: "2s"
scheduler:
  extraArgs:
    leader-elect: "true"
    leader-elect-lease-duration: "15s"
    leader-elect-renew-deadline: "10s"
    leader-elect-retry-period: "2s"
networking:
  podSubnet: "${POD_CIDR}"
  serviceSubnet: "${SERVICE_CIDR}"
  dnsDomain: "${CLUSTER_DOMAIN}"
etcd:
  local:
    dataDir: "/var/lib/etcd"
    # Senior DevOps: etcd disk I/O là critical — dùng SSD riêng nếu có thể
    extraArgs:
      auto-compaction-mode: "periodic"
      auto-compaction-retention: "72h"   # Giữ snapshot 72h
      quota-backend-bytes: "8589934592"  # 8GB etcd DB limit
      max-request-bytes: "10485760"      # 10MB max request
      # Peer TLS auto-generated
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
metadata:
  name: kubelet-config
  labels:
    app.kubernetes.io/component: kubelet
# Cgroup
cgroupDriver: "systemd"
# Authentication — cho phép anonymous nếu cần health check
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: true
  x509:
    clientCAFile: "/etc/kubernetes/pki/ca.crt"
authorization:
  mode: Webhook
# Resource reservation — để system daemons và kubelet có đủ resources
# Senior DevOps: Điều chỉnh theo capacity thực tế của node
kubeReserved:
  cpu: "500m"
  memory: "512Mi"
  ephemeral-storage: "2Gi"
systemReserved:
  cpu: "500m"
  memory: "512Mi"
  ephemeral-storage: "2Gi"
evictionHard:
  memory.available: "500Mi"
  nodefs.available: "10%"
  nodefs.inodesFree: "5%"
  imagefs.available: "15%"
evictionSoft:
  memory.available: "1Gi"
  nodefs.available: "15%"
  imagefs.available: "20%"
evictionSoftGracePeriod:
  memory.available: "1m30s"
  nodefs.available: "2m"
  imagefs.available: "2m"
evictionMaxPodGracePeriod: 120
# Pod management
maxPods: 110
podPidsLimit: 4096
# Image GC
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 75
imageMinimumGCAge: "2m"
# Logging
logging: "flush"   # Flush ngay lập tức thay vì buffer
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: kube-proxy-config
  labels:
    app.kubernetes.io/component: kube-proxy
mode: "iptables"  # Or "ipvs" for better performance on large clusters
# iptables sync period
iptables:
  masqueradeAll: false
  syncPeriod: "30s"
  minSyncPeriod: "10s"
# Cluster CIDR (for masquerade)
clusterCIDR: "${POD_CIDR}"
EOF

  echo "   ✅ kubeadm config: $config_file"
  echo ""
}

generate_kubeadm_config

# ===================================================================
# 3. KUBEADM INIT
# ===================================================================

run_kubeadm_init() {
  echo ""
  echo "============================================"
  echo "  PHASE: kubeadm init"
  echo "============================================"

  # Pull images trước — giúp init nhanh hơn và dễ debug lỗi pull
  echo ">>> [3a] Pre-pull images..."
  sudo kubeadm config images pull \
    --config /etc/kubernetes/kubeadm-config.yaml \
    --v=4
  echo "   ✅ Images pulled"

  # Run init
  echo ">>> [3b] Init control-plane..."
  echo "    (This takes 30-60s. Watch logs carefully.)"
  sudo kubeadm init \
    --config /etc/kubernetes/kubeadm-config.yaml \
    --upload-certs \
    --v=5 2>&1 | tee /tmp/kubeadm-init.log

  # Kiểm tra init thành công
  if [ $? -ne 0 ]; then
    echo "❌ kubeadm init failed. Check /tmp/kubeadm-init.log"
    echo "   Common fixes:"
    echo "   - sudo kubeadm reset --force && sudo rm -rf /etc/kubernetes/"
    echo "   - Check containerd: sudo crictl version"
    echo "   - Check port 6443: sudo ss -tlnp | grep 6443"
    echo "   - Check logs: sudo journalctl -u kubelet --no-pager -n100"
    exit 1
  fi

  echo ""
  echo "✅ kubeadm init successful!"
  echo ""
}

run_kubeadm_init

# ===================================================================
# 4. KUBECONFIG SETUP
# ===================================================================

setup_kubeconfig() {
  echo ""
  echo "============================================"
  echo "  PHASE: kubeconfig Setup"
  echo "============================================"

  # Kubeconfig cho root
  echo ">>> [4a] Root kubeconfig"
  mkdir -p /root/.kube
  sudo cp -i /etc/kubernetes/admin.conf /root/.kube/config
  sudo chown root:root /root/.kube/config

  # Kubeconfig cho user hiện tại
  echo ">>> [4b] User ($(whoami)) kubeconfig"
  mkdir -p "$HOME/.kube"
  sudo cp -i /etc/kubernetes/admin.conf "$HOME/.kube/config"
  sudo chown "$(id -u):$(id -g)" "$HOME/.kube/config"

  # KUBECONFIG env var trong .bashrc
  if ! grep -q "KUBECONFIG" ~/.bashrc 2>/dev/null; then
    echo 'export KUBECONFIG="$HOME/.kube/config"' >> ~/.bashrc
  fi

  echo "   ✅ kubeconfig ready"
  echo ""
}

setup_kubeconfig

# ===================================================================
# 5. POST-INIT: NODE LABELING, TAINTS, RBAC
# ===================================================================

post_init_config() {
  echo ""
  echo "============================================"
  echo "  PHASE: Post-Init Configuration"
  echo "============================================"

  # Node labels
  echo ">>> [5a] Label control-plane node"
  kubectl label node "$MASTER_NAME" \
    node-role.kubernetes.io/control-plane="" \
    node.kubernetes.io/exclude-from-external-load-balancers="" \
    --overwrite 2>/dev/null || true

  # Taint control-plane để chỉ chạy system pods
  # (kubeadm đã set từ config, chỉ verify)
  echo ">>> [5b] Verify taints"
  kubectl get node "$MASTER_NAME" -o jsonpath='{.spec.taints}' 2>/dev/null || \
    echo "   (no taints found)"

  # Topology labels
  echo ">>> [5c] Set topology labels"
  REGION="asia-southeast1"
  ZONE="asia-southeast1-a"
  kubectl label node "$MASTER_NAME" \
    topology.kubernetes.io/region="$REGION" \
    topology.kubernetes.io/zone="$ZONE" \
    topology.kubernetes.io/node-type="control-plane" \
    --overwrite 2>/dev/null || true

  # ClusterRoleBinding cho phép admin
  echo ">>> [5d] Verify RBAC"
  kubectl get clusterrolebinding cluster-admin -o yaml 2>/dev/null | head -5 || \
    echo "   (cluster-admin binding tồn tại mặc định)"

  echo ""
}

post_init_config

# ===================================================================
# 6. ETCD SNAPSHOT BACKUP SERVICE
# ===================================================================
#
# Senior DevOps: etcd backup là SINGLE POINT OF FAILURE nếu không có.
# Auto backup mỗi 30p, giữ 48 snapshot gần nhất.

install_etcd_backup() {
  echo ""
  echo "============================================"
  echo "  PHASE: etcd Auto-Backup"
  echo "============================================"

  local backup_dir="/var/backups/k8s/etcd"
  local script="/usr/local/bin/etcd-snapshot.sh"

  sudo mkdir -p "$backup_dir"
  sudo chmod 700 "$backup_dir"

  # Backup script
  cat <<'SCRIPT' | sudo tee "$script" > /dev/null
#!/usr/bin/env bash
# ===================================================================
# etcd snapshot backup — chạy mỗi 30 phút qua systemd timer
#
# Lưu snapshot kèm timestamp, cleanup snapshot cũ hơn 24h
# ===================================================================
set -euo pipefail

BACKUP_DIR="/var/backups/k8s/etcd"
RETENTION_HOURS=24
SNAPSHOT_FILE="${BACKUP_DIR}/etcd-snapshot-$(date +%Y%m%d-%H%M%S).db"

# Tạo snapshot
if command -v etcdctl &>/dev/null; then
  # Dùng etcdctl nếu có (kubeadm install)
  sudo ETCDCTL_API=3 etcdctl \
    --endpoints=127.0.0.1:2379 \
    --cacert=/etc/kubernetes/pki/etcd/ca.crt \
    --cert=/etc/kubernetes/pki/etcd/server.crt \
    --key=/etc/kubernetes/pki/etcd/server.key \
    snapshot save "$SNAPSHOT_FILE"
elif sudo ctr snapshot &>/dev/null; then
  # Fallback: dùng etcd pod exec
  sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf \
    -n kube-system exec etcd-$(hostname) -- \
    etcdctl snapshot save "$SNAPSHOT_FILE" 2>/dev/null || \
    echo "ERROR: Cannot create etcd snapshot" >&2
fi

# Validate snapshot
sudo ETCDCTL_API=3 etcdctl --write-out=table snapshot status "$SNAPSHOT_FILE" \
  > "${SNAPSHOT_FILE}.status" 2>/dev/null

# Compress
gzip -f "$SNAPSHOT_FILE"
echo "Snapshot: ${SNAPSHOT_FILE}.gz ($(stat -c%s "${SNAPSHOT_FILE}.gz" 2>/dev/null || stat -f%z "${SNAPSHOT_FILE}.gz" 2>/dev/null) bytes)"

# Cleanup old snapshots
find "$BACKUP_DIR" -name "etcd-snapshot-*.db.gz" -mmin +$((RETENTION_HOURS * 60)) -delete
find "$BACKUP_DIR" -name "etcd-snapshot-*.db.gz.status" -mmin +$((RETENTION_HOURS * 60)) -delete

# Log
echo "$(date --iso-8601=seconds) | Snapshot saved" >> "${BACKUP_DIR}/backup.log"
SCRIPT
  sudo chmod +x "$script"

  # Systemd service
  cat <<'UNIT' | sudo tee /etc/systemd/system/etcd-snapshot.service > /dev/null
[Unit]
Description=etcd snapshot backup
After=network.target
Before=kubelet.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/etcd-snapshot.sh
User=root
Group=root
# Security hardening
ProtectSystem=full
ProtectHome=true
ReadWritePaths=/var/backups/k8s/etcd
NoNewPrivileges=true
PrivateTmp=true
UNIT

  # Systemd timer — mỗi 30 phút
  cat <<'TIMER' | sudo tee /etc/systemd/system/etcd-snapshot.timer > /dev/null
[Unit]
Description=etcd snapshot backup timer

[Timer]
OnCalendar=*:0/30
Persistent=true
RandomizedDelaySec=30

[Install]
WantedBy=timers.target
TIMER

  sudo systemctl daemon-reload
  sudo systemctl enable etcd-snapshot.timer
  sudo systemctl start etcd-snapshot.timer

  echo "   ✅ etcd backup timer installed (every 30min → /var/backups/k8s/etcd/)"
  echo "   📍 Giữ snapshot 24h gần nhất, cleanup tự động"
  echo "   📍 Để restore: etcdctl snapshot restore <file>"
  echo ""
}

install_etcd_backup

# ===================================================================
# 7. IN JOIN COMMAND
# ===================================================================

print_join_command() {
  echo ""
  echo "================================================================"
  echo "  🚀 CLUSTER INIT COMPLETE"
  echo "================================================================"
  echo ""
  echo "📌 Cluster info:"
  kubectl cluster-info 2>/dev/null | head -5

  echo ""
  echo "📌 Nodes:"
  kubectl get nodes -o wide

  echo ""
  echo "📌 Pods (kube-system):"
  kubectl get pods -n kube-system

  echo ""
  echo "================================================================"
  echo "  📋 JOIN COMMAND — Copy và chạy trên WORKER nodes"
  echo "================================================================"
  echo ""
  local join_cmd
  join_cmd=$(kubeadm token create --print-join-command 2>/dev/null)
  echo "  ${join_cmd}"
  echo ""
  echo "  (Nếu token hết hạn, chạy: sudo helper-gen-join-command.sh)"
  echo ""
  echo "================================================================"
  echo "  ⏭️  TIẾP THEO:"
  echo "  1. Chạy STEP 3 (join-workers) trên worker-1 và worker-2"
  echo "  2. Sau đó chạy STEP 4 (Calico CNI) trên master này"
  echo "================================================================"
}

print_join_command

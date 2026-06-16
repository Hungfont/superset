# 🔧 Production Kubernetes Cluster Setup — kubeadm + Calico

> **Senior DevOps — 10 years exp. edition**
>
> 3-node cluster, Ubuntu 22.04, Kubernetes 1.32, containerd, Calico VXLAN.
> Production-ready với audit logging, etcd auto-backup, kernel tuning, security hardening.
> **Full stack:** Helm + Ingress-Nginx + MetalLB (LoadBalancer cho bare-metal).
>
> **Automation:** Ansible playbook — một lệnh deploy toàn bộ cluster.

---

## 📐 Kiến trúc

```
┌────────────────────┐     ┌────────────────────┐     ┌────────────────────┐
│    k8s-master      │     │   k8s-worker-1     │     │   k8s-worker-2     │
│  192.168.1.100     │     │  192.168.1.101     │     │  192.168.1.102     │
│  (control-plane)   │     │  (worker)          │     │  (worker)          │
│  etcd, API,        │     │  containerd        │     │  containerd        │
│  scheduler, c-mgr  │     │  kubelet, calico   │     │  kubelet, calico   │
└─────────┬──────────┘     └──────────┬─────────┘     └──────────┬─────────┘
          │                           │                          │
          └───────────────────────────┼──────────────────────────┘
                                      │
                          [  Switch / LAN  ]
```

### Network

| Layer | CIDR | Ghi chú |
|-------|------|---------|
| **Node LAN** | `192.168.1.0/24` | Mạng vật lý |
| **Pod Network** | `10.244.0.0/16` | Calico VXLAN |
| **Service Network** | `10.96.0.0/12` | K8s internal |

---

## 🚀 Quick Start

### Option A: Ansible Automation (recommended — 1 command)

```bash
# Yêu cầu: control machine có Ansible + SSH key access đến cả 3 node

# 1. Sửa inventory cho đúng IP/hostname
vi ansible/inventory/hosts.yml

# 2. Deploy toàn bộ cluster — một lệnh duy nhất
./setup-cluster.sh

# 3. Hoặc chạy không cần xác nhận:
./setup-cluster.sh --yes

# Chạy từng phase riêng:
./setup-cluster.sh --tags bootstrap        # Chỉ bootstrap
./setup-cluster.sh --tags addons           # Chỉ Calico+Ingress+MetalLB
./setup-cluster.sh --check                 # Dry-run trước
./setup-cluster.sh --skip-reboot           # Không auto reboot
```

### Option B: Manual (từng bước)

### Bước 0 — Pre-config (cả 3 node)

Trước tiên, sửa IP/hostname trong các scripts cho đúng môi trường của bạn.

```bash
# File cần sửa:
# - step1-prepare-all-nodes.sh:    HOSTNAME, IP_ADDR
# - step2-init-master.sh:          MASTER_IP, MASTER_NAME
# - step3-join-workers.sh:         HOSTNAME, IP_ADDR, JOIN command
# - helper-setup-hosts.sh:         IP của 3 node
# - helper-open-firewall.sh:       (tự detect)
```

```bash
# ============================================
# 1. /etc/hosts — cả 3 node
# ============================================
bash helper-setup-hosts.sh

# ============================================
# 2. Firewall — cả 3 node (master mở thêm port etcd)
# ============================================
bash helper-open-firewall.sh

# ============================================
# 3. Node bootstrap — cả 3 node (SỬA hostname + IP)
# ============================================
vim step1-prepare-all-nodes.sh
# Sửa: HOSTNAME="k8s-master" / "k8s-worker-1" / "k8s-worker-2"
# Sửa: IP_ADDR="192.168.1.10X"
sudo bash step1-prepare-all-nodes.sh

# ============================================
# 4. REBOOT — cả 3 node
# ============================================
sudo reboot

# ============================================
# 5. kubeadm init — CHỈ master
# ============================================
sudo bash step2-init-master.sh
# 📌 LƯU câu lệnh JOIN từ output

# ============================================
# 6. Join workers — worker-1, worker-2
# ============================================
# Sửa câu lệnh join trong step3-join-workers.sh
sudo bash step3-join-workers.sh

# ============================================
# 7. Calico CNI — master
# ============================================
# Đầu tiên review manifest:
bash step4-install-calico.sh  # dry-run, xem manifest
# Sau đó apply thật:
APPLY_IMMEDIATELY=true sudo bash step4-install-calico.sh

# ============================================
# 8. Verify — master
# ============================================
bash helper-verify-cluster.sh

# ============================================
# 9. Helm — master
# ============================================
bash step5-helm.sh

# ============================================
# 10. Ingress-Nginx — master
# ============================================
bash step6-ingress-nginx.sh

# ============================================
# 11. MetalLB + LoadBalancer — master
# ============================================
# SỬA LB_IP_RANGE trong script cho đúng mạng LAN
bash step7-metallb.sh

# ============================================
# 12. E2E Test — master
# ============================================
bash step8-end-to-end-test.sh
```

---

## 📋 File Structure

```
docs/setup-k8s/
│
├── README.md                          ← File này
│
├── step1-prepare-all-nodes.sh         ← Production node bootstrap
│   ├── Preflight validation
│   ├── Kernel tuning (net.core.somaxconn, vm.*, fs.*)
│   ├── containerd with registry mirrors
│   ├── kubelet pinning & bash aliases
│   └── Final validation
│
├── step2-init-master.sh               ← Production control-plane
│   ├── kubeadm ConfigFile (YAML)
│   ├── Audit logging policy
│   ├── etcd auto-backup (systemd timer, 30 phút)
│   ├── Node labeling & taints
│   └── Resource reservation
│
├── step3-join-workers.sh              ← Join workers
│
├── step4-install-calico.sh            ← Calico production config
│   ├── VXLAN mode (better perf)
│   ├── Typha for scale
│   ├── Felix tuning (log rate-limit)
│   ├── Resource requests/limits
│   └── Default-deny NetworkPolicy
│
├── step5-helm.sh                      ← Helm 3 (repos, plugins, RBAC, aliases)
│
├── step6-ingress-nginx.sh             ← Ingress Controller production
│   ├── Multiple replicas + PDB + HPA
│   ├── Proxy buffer/body tuning
│   ├── SSL/TLS hardening
│   ├── JSON access logs
│   ├── Default SSL cert
│   └── NodeSelector worker nodes only
│
├── step7-metallb.sh                   ← MetalLB LoadBalancer
│   ├── kube-proxy strict ARP
│   ├── IP pool config (L2)
│   ├── BGP config (optional)
│   └── Auto-test LoadBalancer IP
│
├── step8-end-to-end-test.sh           ← E2E test (Ingress + MetalLB + app)              ← /etc/hosts
├── helper-open-firewall.sh            ← UFW rules
├── helper-gen-join-command.sh         ← Token refresh
├── helper-verify-cluster.sh           ← Health check
├── helper-reset-worker.sh             ← Reset worker
├── calico-patch-ip-autodetect.sh      ← Fix Calico IP detection
│
├── setup-cluster.sh                   ← 🚀 **ONE COMMAND** — Ansible wrapper
│
├── ansible/
│   ├── ansible.cfg                    ← Production Ansible config
│   ├── requirements.yml               ← Collections (kubernetes.core, etc.)
│   ├── site.yml                       ← Main playbook (6 phases)
│   ├── inventory/
│   │   └── hosts.yml                  ← Inventory — SỬA IP TẠI ĐÂY
│   ├── group_vars/
│   │   └── all.yml                    ← Global variables
│   └── roles/
│       ├── common/                    ← APT, /etc/hosts, base packages
│       ├── bootstrap/                 ← Kernel, containerd, kube* packages
│       ├── control-plane/             ← kubeadm init, audit, etcd backup
│       ├── worker/                    ← kubeadm join
│       └── addons/                    ← Calico, Helm, Ingress-Nginx, MetalLB
│
└── production/
    ├── README.md                      ← Production ops index
    ├── security-hardening.md          ← PSA, seccomp, RBAC, CIS, AppArmor
    ├── node-maintenance.md            ← Drain, patch, upgrade strategy
    ├── monitoring-observability.md    ← Metrics, logs, alerts
    └── etcd-restore-and-recovery.md   ← Disaster recovery
```

---

## 🛡️ Production Features

| Feature | Config | File |
|---------|--------|------|
| **Kernel tuning** | somaxconn=32768, conntrack=2M, swappiness=0 | step1 |
| **Registry mirror** | Cho registry.k8s.io tránh rate-limit | step1 |
| **Audit logging** | Policy đầy đủ, log secrets access | step2 |
| **etcd backup** | Auto snapshot mỗi 30p, giữ 24h | step2 |
| **Resource reservation** | kubeReserved + systemReserved | step2 |
| **Calico VXLAN** | Better throughput, cross-subnet | step4 |
| **Typha** | Scale cho API server | step4 |
| **Pod Security** | baseline enforcement | production/security |
| **Seccomp** | RuntimeDefault profile | production/security |
| **Default deny** | NetworkPolicy cho system NS | step4 |
| **Node monitoring** | Health script + cron | production/monitoring |
| **Metrics-Server** | `kubectl top` support | production/monitoring |
| **Node-Problem-Detector** | Phát hiện kernel/disk issues | production/monitoring |
| **CoreDNS autoscaling** | Cluster-proportional-autoscaler | production/monitoring |
| **DR plan** | etcd restore procedures | production/etcd |
| **Helm 3** | Repos, plugins, RBAC, OCI support | step5 |
| **Ingress-Nginx** | HA (HPA+PDB), SSL hardening, JSON logs, rate-limit | step6 |
| **MetalLB (L2/BGP)** | LoadBalancer cho bare-metal, IP pool, strict ARP | step7 |
| **E2E Test** | Ingress + MetalLB + app verification | step8 |

---

## 🔥 Troubleshooting Production

| Triệu chứng | Root cause | Fix |
|-------------|-----------|-----|
| Node `NotReady` sau Calico | IP auto-detection sai (multi-NIC) | `bash calico-patch-ip-autodetect.sh` |
| CoreDNS `CrashLoopBackOff` | DNS query quá tải / OOM | Tăng resources: `kubectl -n kube-system edit deploy/coredns` |
| `kubectl top` not working | metrics-server chưa cài | `kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml` |
| Pod stuck `ContainerCreating` | CNI config chưa apply | `kubectl -n calico-system logs daemonset/calico-node` |
| API server slow | etcd disk I/O cao | Kiểm tra: `iostat -x 1` — etcd cần SSD |
| Certificate expiry | Cert tự động 1 năm | `sudo kubeadm certs check-expiration` |
| etcd backup fail | Disk full | `df -h /var/backups/k8s/etcd/` |
| Ingress `503 Service Temporarily Unavailable` | Backend service not ready | `kubectl -n <ns> get endpoints` |
| Ingress không nhận IP | MetalLB chưa cấu hình strict ARP | `kubectl get configmap kube-proxy -n kube-system -o yaml \| grep strictARP` |
| LoadBalancer IP `pending` mãi | MetalLB IP pool sai subnet | Kiểm tra: `kubectl -n metallb-system get ippools` |
| Helm `connection refused` | Repo URL sai / network | `helm repo list && helm repo update` |

---

## 📊 Port Reference

| Port | Protocol | Purpose | Node |
|------|----------|---------|------|
| 6443 | TCP | Kubernetes API Server | Master |
| 2379-2380 | TCP | etcd server + peer | Master |
| 10250 | TCP | Kubelet API | All |
| 10257 | TCP | kube-controller-manager | Master |
| 10259 | TCP | kube-scheduler | Master |
| 30000-32767 | TCP | NodePort services | All |
| 179 | TCP | BGP (Calico) | All |
| 5473 | TCP | Calico Typha | Master |
| 4789 | UDP | VXLAN (Calico) | All |
| 51820-51821 | UDP | WireGuard (Calico) | All |
| 7946 | TCP/UDP | MetalLB memberlist (speaker) | Worker (L2) |
| 7472 | TCP | MetalLB metrics | Worker |
| 80 | TCP | Ingress HTTP | All (khi Ingress dùng host-network) |
| 443 | TCP | Ingress HTTPS | All (khi Ingress dùng host-network) |

---

## 📈 Post-Setup Checklist

### Day 1
- [ ] `bash helper-verify-cluster.sh` — all nodes Ready, all pods Running
- [ ] `kubectl run nginx --image=nginx --port=80` — test deployment
- [ ] `kubectl exec -it nginx -- /bin/sh` — test network (ping, DNS)
- [ ] Kiểm tra etcd backup: `ls -la /var/backups/k8s/etcd/`
- [ ] Kiểm tra audit log: `sudo ls -la /var/log/kubernetes/audit/`
- [ ] `bash step8-end-to-end-test.sh` — verify Ingress + MetalLB

### Day 2
- [ ] Apply security hardening: `production/security-hardening.md`
- [ ] Install metrics-server: `production/monitoring-observability.md`
- [ ] Install node-problem-detector: `production/monitoring-observability.md`
- [ ] Set PodDisruptionBudget cho CoreDNS
- [ ] Deploy test app với Service type LoadBalancer (verify MetalLB)

### Week 1
- [ ] Install Prometheus + Grafana
- [ ] Setup log aggregation (Loki hoặc EFK)
- [ ] RBAC review — xóa cluster-admin bindings không cần thiết
- [ ] Setup Velero cho resource backup
- [ ] NetworkPolicy cho tất cả user namespaces
- [ ] Cấu hình Let's Encrypt (cert-manager) cho Ingress SSL

---

> **Maintenance note:** Thay đổi cấu hình cluster bằng ConfigFile YAML (step2 dùng),
> không dùng flags. ConfigFile có thể check vào git để version control.

# Ansible Automation — Kubernetes Cluster Setup

## Usage

```bash
# 1. Sửa inventory cho đúng IP của bạn
vi inventory/hosts.yml

# 2. Deploy toàn bộ cluster
./setup-cluster.sh

# Hoặc từ thư mục ansible:
cd ansible
ansible-playbook site.yml
```

## Playbook Structure

```
site.yml  ← Main playbook (6 phases)
├── PHASE 1: common      — /etc/hosts, base packages, disable unattended-upgrades
├── PHASE 2: bootstrap   — kernel tuning, containerd, kubeadm/kubelet/kubectl
├── PHASE 3: reboot      — reboot all nodes (sequential)
├── PHASE 4: control-plane — kubeadm init, audit policy, etcd backup
├── PHASE 5: worker      — kubeadm join
└── PHASE 6: addons      — Calico → Helm → Ingress-Nginx → MetalLB
```

## Tags

```bash
ansible-playbook site.yml --tags bootstrap    # Chỉ chạy bootstrap
ansible-playbook site.yml --tags addons       # Chỉ cài addons
ansible-playbook site.yml --tags common       # Chỉ base setup
ansible-playbook site.yml --skip-tags reboot  # Không reboot
```

## Inventory

Edit `inventory/hosts.yml`:

| Variable | Default | Description |
|----------|---------|-------------|
| `pod_network_cidr` | 10.244.0.0/16 | Calico pod network |
| `service_network_cidr` | 10.96.0.0/12 | K8s service network |
| `lb_ip_range` | 192.168.1.200-210 | MetalLB IP pool |
| `kubernetes_version` | 1.32 | K8s version |
| `network_interface` | eth0 | Primary network iface |
| `calico_network_mode` | vxlan | vxlan or ipip |

## Requirements

- **Control machine**: Linux/Mac with Ansible >= 9 (`pip install ansible`)
- **Target nodes**: Ubuntu 22.04, SSH key access as root
- **Network**: All nodes can reach each other, internet access for package downloads

## One-liner setup on control machine

```bash
# Ubuntu control machine
sudo apt update && sudo apt install -y python3-pip sshpass
pip3 install ansible
ansible-galaxy collection install -r requirements.yml

# Test connectivity
ansible all -i inventory/hosts.yml -m ping

# Deploy
ansible-playbook site.yml
```

# Tổng Hợp Kiến Thức Ansible — Kubernetes Cluster Setup

> Tài liệu tổng hợp các Ansible patterns, techniques, best practices đã sử dụng trong playbook K8s.

---

## 1. Kiến Trúc Tổng Thể

```
ansible/
├── ansible.cfg              ← Global config (SSH, caching, output)
├── requirements.yml         ← Collection dependencies
├── site.yml                 ← Main playbook — orchestrator (6 phases)
├── inventory/hosts.yml      ← YAML inventory với group hierarchy
├── group_vars/all.yml       ← Global variables
└── roles/
    ├── common/              ← apt, /etc/hosts, base packages
    ├── bootstrap/           ← kernel, containerd, kubeadm/kubelet/kubectl
    ├── control-plane/       ← kubeadm init, audit, etcd backup
    ├── worker/              ← kubeadm join
    └── addons/              ← Calico, Helm, Ingress-Nginx, MetalLB
```

**Key insight:** `site.yml` là **orchestrator** — không chứa logic, chỉ gọi roles đúng thứ tự. Mỗi role là đơn vị độc lập.

---

## 2. Ansible Core Patterns

### 2.1. `ansible.cfg` — Production Tuning

```ini
[defaults]
inventory                = inventory/hosts.yml
gathering                = smart               # Cache facts
fact_caching             = jsonfile
fact_caching_connection  = /tmp/ansible-facts
fact_caching_timeout     = 86400               # 24h cache

[ssh_connection]
pipelining               = True                # ⭐ Giảm SSH overhead 2-3x
ssh_args                 = -o ControlMaster=auto -o ControlPersist=60s

[defaults]
stdout_callback          = yaml                # Output dễ đọc
callback_whitelist       = profile_tasks       # Show timing
```

**Senior insight:** `pipelining=True` giảm SSH connections từ `n*3` xuống `n*1`.

---

### 2.2. Inventory — YAML với Group Hierarchy

```yaml
all:
  vars:                       # Global — apply mọi host
    pod_network_cidr: "10.244.0.0/16"
    kubernetes_version: "1.32"

  children:
    k8s_cluster:              # Parent group
      children:
        control_plane:        # Child group 1
          hosts:
            k8s-master:
              ansible_host: 192.168.1.100
              node_ip: 192.168.1.100
              node_name: k8s-master

        workers:              # Child group 2
          hosts:
            k8s-worker-1:
              ansible_host: 192.168.1.101
```

**Pattern:** `children` → group hierarchy, `all.vars` → global, `hosts.*` → per-host.

---

### 2.3. Site Playbook — Multi-Phase Orchestrator

```yaml
- name: "PHASE 1: Common"
  hosts: k8s_cluster            # Group name từ inventory
  gather_facts: true
  roles:
    - common
  tags: [common, phase1]

- name: "PHASE 3: Reboot"
  hosts: k8s_cluster
  tasks:
    - name: "Reboot one at a time"
      ansible.builtin.reboot:
        reboot_timeout: 300
      throttle: 1                # ⭐ Reboot từng máy, ko đồng loạt
  tags: [reboot, phase3]
```

**Key techniques:**
- `hosts: k8s_cluster` → target group, not individual host
- `tags` → subset execution: `--tags bootstrap,control-plane`
- `throttle: 1` → control parallelism
- `gather_facts: false` → skip facts for reboot phase

---

### 2.4. Handlers — Restart/Notify Pattern

```yaml
# handlers/main.yml
- name: restart containerd
  ansible.builtin.systemd:
    name: containerd
    state: restarted
    daemon_reload: true

# tasks/main.yml
- name: "Enable SystemdCgroup"
  ansible.builtin.replace:
    path: /etc/containerd/config.toml
    regexp: 'SystemdCgroup = false'
    replace: 'SystemdCgroup = true'
  notify: restart containerd      # ⭐ Chỉ restart khi CÓ thay đổi
```

**Rule:** Handler chỉ chạy khi task thực sự `changed`. Dùng `notify` với `replace` để tránh restart vô ích.

---

### 2.5. Idempotency — "Chạy lại không sao"

| Technique | Example | Purpose |
|-----------|---------|---------|
| `creates` | `kubeadm init` có `creates: /etc/kubernetes/admin.conf` | Skip nếu file tồn tại |
| `stat + when` | `stat: /etc/kubernetes/kubelet.conf` + `when: not stat.exists` | Conditional execution |
| `dpkg_selections` | `selection: hold` cho kubelet | Version pinning |
| `replace` | `regexp + replace` trên fstab | Idempotent regex |
| `lineinfile` | Thêm 1 dòng nếu chưa có | Single-line idempotency |

---

### 2.6. Variables & Jinja2 Templates

```yaml
# group_vars/all.yml
kernel_params:
  net.core.somaxconn: 32768
  vm.swappiness: 0
```

```yaml
# tasks — loop dict
- ansible.posix.sysctl:
    name: "{{ item.key }}"
    value: "{{ item.value }}"
  loop: "{{ kernel_params | dict2items }}"
```

```jinja2
{# kubeadm-config.yaml.j2 — template #}
nodeRegistration:
  name: "{{ node_name }}"
  kubeletExtraArgs:
    node-ip: "{{ node_ip }}"
networking:
  podSubnet: "{{ pod_network_cidr }}"
```

---

### 2.7. Delegation — Master → Worker Communication

```yaml
# Master: tạo token, lưu local
- name: "Save join command"
  ansible.builtin.copy:
    content: "{{ k8s_join_command }}"
    dest: /tmp/k8s-join-command.txt
  delegate_to: localhost      # ⭐ Trên control machine
  run_once: true

# Worker: đọc token từ master
- name: "Read join command"
  ansible.builtin.slurp:
    src: /tmp/k8s-join-command.txt
  register: join_cmd_file
  delegate_to: "{{ groups['control_plane'][0] }}"  # ⭐ Trên master
  run_once: true
```

---

### 2.8. Wait & Retry Patterns

```yaml
# Simple pause
- ansible.builtin.pause:
    seconds: 20

# Retry until condition
- name: "Wait for LoadBalancer IP"
  ansible.builtin.shell: "kubectl get svc ... -o jsonpath='{.status...}'"
  register: result
  retries: 10                # 10 lần
  delay: 6                   # mỗi lần 6s
  until: result.stdout != ""

# Safe shell (ko fail playbook)
- name: "Check Calico status"
  ansible.builtin.shell: "kubectl rollout status ds/calico-node --timeout=180s"
  failed_when: false
  changed_when: false
```

---

### 2.9. Kubernetes Native Modules

| Module | Usage | Key Options |
|--------|-------|-------------|
| `kubernetes.core.k8s` | CRD, Node, ConfigMap, Secret | `state: patched`, `src: file.yaml` |
| `kubernetes.core.helm` | Deploy Helm chart | `values: {...}`, `wait: true`, `atomic: true` |
| `kubernetes.core.k8s_info` | Read K8s state | `api_version`, `kind`, `name` |

```yaml
- kubernetes.core.helm:
    name: ingress-nginx
    chart_ref: ingress-nginx/ingress-nginx
    values:
      controller:
        replicaCount: 2
        service:
          type: LoadBalancer
    wait: true
    timeout: "10m"
    atomic: true              # ⭐ Rollback nếu fail
```

---

### 2.10. Collections

```yaml
# requirements.yml
collections:
  - name: community.general     # timezone, modprobe
    version: ">=9.0.0"
  - name: kubernetes.core       # k8s, helm
    version: ">=3.0.0"
```

```bash
ansible-galaxy collection install -r requirements.yml
```

---

## 3. Jinja2 Filters Cheat Sheet

| Filter | Example | Purpose |
|--------|---------|---------|
| `dict2items` | `{{ kernel_params \| dict2items }}` | Dict → list for loop |
| `b64decode` | `{{ content \| b64decode }}` | Decode slurp content |
| `b64encode` | `{{ cert \| b64encode }}` | Encode for K8s Secret |
| `select('match')` | `lines \| select('match', '^kubeadm')` | Filter output |
| `default` | `result \| default('unknown')` | Fallback nếu undefined |
| Ternary | `{{ 'VXLAN' if mode == 'vxlan' else 'IPIP' }}` | Conditional |

---

## 4. Useful Commands

```bash
# Run subset
ansible-playbook site.yml --tags bootstrap,control-plane

# Skip
ansible-playbook site.yml --skip-tags reboot,addons

# Dry run
ansible-playbook site.yml --check

# Verbose
ansible-playbook site.yml -vvv

# Limit to specific hosts
ansible-playbook site.yml --limit workers

# Ad-hoc commands
ansible all -i hosts.yml -m ping -o
ansible workers -m shell -a "kubectl get nodes"
ansible control_plane -m shell -a "kubeadm token create --print-join-command"

# Ansible Vault (encrypt sensitive vars)
ansible-vault encrypt group_vars/all.yml
ansible-playbook site.yml --ask-vault-pass

# Gather system info
ansible all -m setup --tree /tmp/facts
```

---

## 5. Senior DevOps Tips

| # | Tip | Why |
|---|-----|-----|
| 1 | `pipelining=True` | Giảm SSH overhead 40% |
| 2 | Idempotent là luật bất biến | Chạy lại ko lỗi |
| 3 | `creates` > `when` + `stat` | Built-in skip, cleaner |
| 4 | Handler chỉ notify khi changed | Ko restart service vô ích |
| 5 | `throttle: 1` cho reboot | Ko reboot đồng loạt |
| 6 | `delegate_to: localhost` cho secrets | Lưu token trên control machine |
| 7 | Template > Copy cho config động | Jinja2 > raw file |
| 8 | `group_vars/` > inline vars | Tách biệt code và config |
| 9 | K8s modules > shell | Ansible quản lý state |
| 10 | `--tags` / `--skip-tags` | Ko cần sửa playbook để run subset |

---

## 6. Execution Flow (6 Phases)

```
./setup-cluster.sh
  ├── Preflight: ping, check Ansible, inventory
  │
  └── ansible-playbook site.yml
        │
        ├── [PHASE 1] common ──── apt, /etc/hosts, disable auto-update
        ├── [PHASE 2] bootstrap ── kernel, containerd, kubeadm/kubectl
        ├── [PHASE 3] reboot ───── reboot từng node
        ├── [PHASE 4] control-plane ── kubeadm init, audit, etcd backup
        ├── [PHASE 5] worker ──── kubeadm join
        └── [PHASE 6] addons ──── Calico → Helm → Ingress → MetalLB
```

Bỏ qua phase 6 nếu chỉ muốn K8s + Calico:

```bash
ansible-playbook site.yml --skip-tags addons,phase6
```

# etcd Disaster Recovery

## Prerequisites

```bash
# etcdctl được cài khi chạy kubeadm
# Nếu chưa có, cài:
kubectl -n kube-system exec etcd-$(hostname) -- etcdctl version
```

## Backup (auto — timer mỗi 30 phút)

Backup được lưu tại: `/var/backups/k8s/etcd/`

```bash
ls -lh /var/backups/k8s/etcd/
# etcd-snapshot-20250101-143000.db.gz
# etcd-snapshot-20250101-143000.db.gz.status
```

## Restore Procedures

### Scenario A: Single control-plane node failure (etcd data corrupt)

```bash
# 1. Stop kubelet và containers
sudo systemctl stop kubelet
sudo crictl stop $(sudo crictl ps --name etcd -q) 2>/dev/null || true

# 2. Backup current data (defense)
sudo mv /var/lib/etcd /var/lib/etcd.corrupted.$(date +%s)

# 3. Restore từ snapshot gần nhất
SNAPSHOT=$(ls -t /var/backups/k8s/etcd/etcd-snapshot-*.db.gz | head -1)
gunzip -k -f "$SNAPSHOT"
sudo ETCDCTL_API=3 etcdctl snapshot restore "${SNAPSHOT%.gz}" \
  --data-dir=/var/lib/etcd \
  --name=$(hostname) \
  --initial-cluster=$(hostname)=https://127.0.0.1:2380 \
  --initial-cluster-token=etcd-cluster \
  --initial-advertise-peer-urls=https://127.0.0.1:2380

# 4. Start kubelet
sudo systemctl start kubelet

# 5. Verify
kubectl get nodes
kubectl get pods -A
```

### Scenario B: Cluster control-plane dead (3+ nodes HA — not covered here)

For HA control-plane with stacked etcd, restore from backup on healthy member.

### Scenario C: Full cluster loss (all 3 nodes down)

```bash
# Trên master:
# 1. Fresh OS install + STEP 1, STEP 2
# 2. Stop kubelet sau khi init (chưa apply Calico)
sudo systemctl stop kubelet
sudo rm -rf /var/lib/etcd

# 3. Restore snapshot
SNAPSHOT=$(ls -t /tmp/etcd-backup/*.db.gz | head -1)
gunzip -k -f "$SNAPSHOT"
sudo ETCDCTL_API=3 etcdctl snapshot restore "${SNAPSHOT%.gz}" \
  --data-dir=/var/lib/etcd

# 4. Start kubelet
sudo systemctl start kubelet

# 5. Verify
kubectl get nodes
kubectl get pods -A

# 6. Rejoin workers (reset + join lại)
```

### Scenario D: Partial data loss (some resources deleted)

```bash
# Nếu chỉ mất resources (không mất etcd):
# Backup resources bằng Velero hoặc:
kubectl get all --all-namespaces -o yaml > /tmp/k8s-all-resources-$(date +%s).yaml
```

## Recovery Checklist

- [ ] Xác định scope: single node hay full cluster
- [ ] Verify snapshot file còn nguyên vẹn: `file etcd-snapshot-*.db`
- [ ] Check snapshot status: `etcdctl snapshot status etcd-snapshot-*.db`
- [ ] Backup hiện tại trước khi restore
- [ ] Sau restore: verify all nodes, pods, services
- [ ] Check workload không bị mất data

# Production Scripts — Cluster Operations

Thư mục này chứa các tài liệu và scripts cho production operations.

## Tổng quan

```
production/
├── README.md                         ← File này
├── etcd-restore-and-recovery.md      ← etcd DR plan
├── node-maintenance.md               ← Drain, upgrade, patch strategy
├── security-hardening.md             ← PSA, seccomp, RBAC, CIS, NetworkPolicy
└── monitoring-observability.md       ← Metrics, logs, alerts, health checks
```

## Priority cho production

### Tuần 1 (bắt buộc ngay sau setup)

1. **Security Hardening** — PSA, RBAC, default-deny NetworkPolicy
2. **Metrics-Server** — `kubectl top` cho resource monitoring
3. **Node-Problem-Detector** — phát hiện sớm node issues
4. **etcd backup verification** — kiểm tra backup đang chạy

### Tuần 2 (nên làm)

5. **Prometheus + Grafana** — full monitoring stack
6. **CoreDNS autoscaling** — đảm bảo DNS không bị quá tải
7. **PodDisruptionBudget** cho critical workloads

### Tháng 1 (tối ưu hóa)

8. **Velero** — backup/restore resources và PV
9. **Descheduler** — pod balancing
10. **Goldilocks** — resource recommendations

## Incident Response

| Incident | Runbook |
|----------|---------|
| Node NotReady | `node-maintenance.md` → drain + investigate |
| etcd corrupt | `etcd-restore-and-recovery.md` |
| Pod CrashLoop | `kubectl describe pod`, `kubectl logs` |
| Disk full | `monitoring-observability.md` → node-health-check |

## Cluster Info Quick Reference

```bash
# Lưu thông tin cluster để tham khảo
CLUSTER_INFO_FILE="/root/cluster-info.txt"
{
  echo "=== Cluster Info ==="
  echo "Date: $(date --iso-8601=seconds)"
  echo ""
  echo "--- Nodes ---"
  kubectl get nodes -o wide
  echo ""
  echo "--- K8s Version ---"
  kubectl version --short
  echo ""
  echo "--- Network ---"
  echo "Pod CIDR: 10.244.0.0/16"
  echo "Service CIDR: 10.96.0.0/12"
  echo "CNI: Calico VXLAN"
  echo ""
  echo "--- Storage ---"
  kubectl get storageclass
} > "$CLUSTER_INFO_FILE"
echo "Cluster info saved to $CLUSTER_INFO_FILE"
```

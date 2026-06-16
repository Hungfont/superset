# Monitoring & Observability — Production Baseline

## 1. Metrics-Server (Resource Metrics)

Cần cho `kubectl top`, HPA, và `kubectl describe node` resource usage:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Senior DevOps: Trong production, metrics-server cần kubelet-certificate-authority
# Nếu dùng kubeadm self-signed certs, cần patch:
kubectl patch deployment metrics-server -n kube-system --type='json' \
  -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--kubelet-preferred-address-types=InternalIP,Hostname,InternalDNS,ExternalDNS,ExternalIP"
  }]'

# Verify
kubectl -n kube-system rollout status deployment metrics-server
kubectl top nodes
kubectl top pods -A
```

## 2. Node Problem Detector

Phát hiện kernel issues, disk errors, network problems:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/node-problem-detector/master/deploy/node-problem-detector-daemonset.yaml

# Kiểm tra
kubectl -n kube-system get pods -l app=node-problem-detector
kubectl -n kube-system logs -l app=node-problem-detector --tail=20
```

## 3. CoreDNS Autoscaling

CoreDNS cần scale theo số lượng pods và nodes:

```bash
# Cluster Proportional Autoscaler cho CoreDNS
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/cluster-proportional-autoscaler/master/artifacts/cluster-proportional-autoscaler.yaml

# Kiểm tra config
kubectl -n kube-system get configmap coredns-autoscaler -o yaml
```

## 4. Node Resource Usage Alerting (Script)

```bash
cat <<'SCRIPT' | sudo tee /usr/local/bin/node-health-check.sh
#!/usr/bin/env bash
# ===================================================================
# Node health check script — chạy cron mỗi 5 phút
# Gửi cảnh báo nếu node có vấn đề
# ===================================================================
set -euo pipefail

ALERT_THRESHOLD_DISK=85      # % disk usage
ALERT_THRESHOLD_MEM=90       # % memory usage
ALERT_THRESHOLD_CPU=80       # % CPU usage
SLACK_WEBHOOK=""             # Set nếu muốn Slack notification

# Node info
NODE=$(hostname)
DATE=$(date --iso-8601=seconds)

# Disk
DISK_USAGE=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -gt "$ALERT_THRESHOLD_DISK" ]; then
  echo "[$DATE] WARNING: Disk usage ${DISK_USAGE}% on ${NODE}"
  # Alert here (Slack, email, webhook)
fi

# Memory
MEM_TOTAL=$(free -m | awk '/^Mem:/ {print $2}')
MEM_USED=$(free -m | awk '/^Mem:/ {print $3}')
MEM_PCT=$((MEM_USED * 100 / MEM_TOTAL))
if [ "$MEM_PCT" -gt "$ALERT_THRESHOLD_MEM" ]; then
  echo "[$DATE] WARNING: Memory usage ${MEM_PCT}% on ${NODE}"
fi

# Kubelet status
if ! systemctl is-active --quiet kubelet; then
  echo "[$DATE] CRITICAL: kubelet not running on ${NODE}"
fi

# containerd status
if ! systemctl is-active --quiet containerd; then
  echo "[$DATE] CRITICAL: containerd not running on ${NODE}"
fi

# Disk I/O pressure (kiểm tra bằng iostat nếu có)
if command -v iostat &>/dev/null; then
  IO_UTIL=$(iostat -x 1 2 | awk '/^avg/ {print $NF}' | tail -1)
  if [ "$(echo "$IO_UTIL > 90" | bc 2>/dev/null)" = "1" ]; then
    echo "[$DATE] WARNING: Disk I/O util ${IO_UTIL}% on ${NODE}"
  fi
fi

# OOM check
dmesg --level=emerg,alert,crit,err 2>/dev/null | grep -i "oom\|killed" | tail -5 && \
  echo "[$DATE] WARNING: OOM events detected on ${NODE}"
SCRIPT
chmod +x /usr/local/bin/node-health-check.sh

# Cron job
(crontab -l 2>/dev/null; echo "*/5 * * * * /usr/local/bin/node-health-check.sh >> /var/log/node-health.log 2>&1") | crontab -
```

## 5. Event Export (Cluster Events)

```bash
# Senior DevOps: Kubernetes events không persistent — cần export
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: event-exporter-config
  namespace: kube-system
data:
  config.yaml: |
    sink: stdout
    filters:
    - type: regex
      regex: ".*(Failed|Killing|Unhealthy|NodeNotReady|BackOff).*"
EOF
```

## 6. Persistent Logging

```bash
# Kiểm tra kubelet logs từ journald
sudo journalctl -u kubelet --no-pager -n 20

# Audit logs (đã config ở STEP 2)
sudo tail -f /var/log/kubernetes/audit/audit.log | jq 'select(.responseStatus.code >= 400)' 2>/dev/null

# Containerd logs
sudo journalctl -u containerd --no-pager -n 20

# Cấu hình log rotation cho audit logs (kubeadm tự động manage qua container)
ls -lh /var/log/kubernetes/audit/
```

## 7. Cluster Resource Dashboard (Commands)

```bash
echo "=== NODE RESOURCES ==="
kubectl top nodes
echo ""
echo "=== POD RESOURCES (top 10 by CPU) ==="
kubectl top pods -A --sort-by=cpu 2>/dev/null | head -11
echo ""
echo "=== POD RESOURCES (top 10 by Memory) ==="
kubectl top pods -A --sort-by=memory 2>/dev/null | head -11
```

## Recommended Production Stack

Đây là những gì cần thêm vào cho production monitoring:

| Component | Chức năng | Ghi chú |
|-----------|-----------|---------|
| **Prometheus Stack** | Full metrics, alerting | kube-prometheus-stack (Helm) |
| **Grafana** | Dashboards | Đi kèm Prometheus stack |
| **Loki / EFK** | Log aggregation | Loki nhẹ hơn EFK |
| **Velero** | Backup resources + volumes | Restore nhanh hơn etcd snapshot |
| **Goldilocks** | Resource recommendations | Dựa trên VPA recommendations |
| **Descheduler** | Pod balancing | Evict pods từ node overloaded |

## Quick Health Dashboard (kubectl one-liners)

```bash
# Cluster overview
alias k8s-dashboard='kubectl get nodes -o wide && echo "" && kubectl get pods -A && echo "" && kubectl top nodes'

# Node resource usage
alias k8s-nodes-full='kubectl get nodes -o custom-columns=NAME:.metadata.name,CPU:.status.capacity.cpu,MEM:.status.capacity.memory,PODS:.status.capacity.pods,K8S:.status.nodeInfo.kubeletVersion,OS:.status.nodeInfo.osImage'

# Check pod distribution
alias k8s-pod-dist='kubectl get pods -A -o wide --sort-by=.spec.nodeName'
```

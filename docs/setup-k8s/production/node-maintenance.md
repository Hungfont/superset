# Node Maintenance — Production Runbook

## 1. Graceful Node Drain (maintenance/reboot)

Cho phép workload di chuyển trước khi reboot node:

```bash
# Drain node (evict pods gracefully)
kubectl drain k8s-worker-1 \
  --ignore-daemonsets \
  --delete-emptydir-data \
  --timeout=120s \
  --grace-period=60

# Kiểm tra node đã cordoned + drained
kubectl get nodes

# Thực hiện maintenance...
sudo apt-get update && sudo apt-get upgrade -y
sudo reboot

# Sau reboot, uncordon
kubectl uncordon k8s-worker-1

# Verify pods running lại
kubectl get pods -o wide | grep k8s-worker-1
```

## 2. Rolling OS Patch Strategy

### 2a. Update một node tại một thời điểm

```bash
# Script drain → patch → reboot → uncordon
NODE="k8s-worker-1"

echo ">>> Draining $NODE..."
kubectl drain "$NODE" --ignore-daemonsets --delete-emptydir-data --timeout=180s

echo ">>> OS update..."
ssh "$NODE" "sudo apt-get update && sudo apt-get upgrade -y"

echo ">>> Reboot..."
ssh "$NODE" "sudo reboot" &
sleep 5

echo ">>> Wait for node to return..."
until kubectl get node "$NODE" 2>/dev/null | grep -q " Ready"; do
  sleep 5
done
sleep 30  # Wait for pods to stabilize

echo ">>> Uncordon..."
kubectl uncordon "$NODE"
```

### 2b. Thứ tự patch

1. **Worker-1** → wait healthy → **Worker-2** → wait → **Master**
2. Master phải patch cuối vì etcd và control-plane

## 3. Node Health Monitoring

```bash
# === Script kiểm tra node health ===
HEALTHY=true

for node in $(kubectl get nodes -o name); do
  n=${node#node/}
  status=$(kubectl get node "$n" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')

  if [ "$status" != "True" ]; then
    echo "❌ $n: NotReady"
    HEALTHY=false
    continue
  fi

  # Check disk pressure
  disk=$(kubectl get node "$n" -o jsonpath='{.status.conditions[?(@.type=="DiskPressure")].status}')
  mem=$(kubectl get node "$n" -o jsonpath='{.status.conditions[?(@.type=="MemoryPressure")].status}')
  pid=$(kubectl get node "$n" -o jsonpath='{.status.conditions[?(@.type=="PIDPressure")].status}')

  [ "$disk" = "True" ] && echo "⚠️  $n: DiskPressure" && HEALTHY=false
  [ "$mem" = "True" ] && echo "⚠️  $n: MemoryPressure" && HEALTHY=false
  [ "$pid" = "True" ] && echo "⚠️  $n: PIDPressure" && HEALTHY=false

  # Node capacity
  echo "$n: Ready"
  kubectl get node "$n" -o jsonpath='{range .status.allocatable}CPU: {.cpu}, Memory: {.memory}, Pods: {.pods}{"\n"}{end}'
done

$HEALTHY && echo "✅ All nodes healthy" || echo "⚠️  Some nodes unhealthy"
```

## 4. Certificate Management

Kubeadm tự động rotate certificates khi kubelet restart:

```bash
# Kiểm tra cert expiry
sudo kubeadm certs check-expiration

# Manual renew (nếu cần)
sudo kubeadm certs renew all

# Sau renew, restart control-plane pods
sudo systemctl restart kubelet
```

## 5. Cluster Upgrade (kubeadm)

```bash
# === Upgrade từ 1.32.x lên 1.32.y (patch) ===

# Master:
sudo apt-get update
sudo apt-get install -y --allow-change-held-packages \
  kubelet=1.32.2-* kubeadm=1.32.2-* kubectl=1.32.2-*
sudo kubeadm upgrade apply v1.32.2
sudo systemctl restart kubelet

# Workers:
sudo apt-get install -y --allow-change-held-packages \
  kubelet=1.32.2-* kubeadm=1.32.2-*
sudo kubeadm upgrade node
sudo systemctl restart kubelet
```

## 6. Pod Disruption Budgets

Sau khi cluster chạy, apply PDB cho critical workloads:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: coredns-pdb
  namespace: kube-system
spec:
  minAvailable: 1
  selector:
    matchLabels:
      k8s-app: kube-dns
```

```bash
kubectl apply -f - <<EOF
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: coredns-pdb
  namespace: kube-system
spec:
  minAvailable: 1
  selector:
    matchLabels:
      k8s-app: kube-dns
EOF
```

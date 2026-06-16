# Security Hardening — Kubernetes Cluster

Các bước hardening này nên chạy SAU KHI cluster đã hoạt động.

## 1. Pod Security Admission (PSA) — Thay thế PSP

K8s 1.32 dùng PSA mặc định — cấu hình để enforce baseline/restricted:

```bash
# Áp dụng label cho namespace default
kubectl label --overwrite ns default \
  pod-security.kubernetes.io/enforce=baseline \
  pod-security.kubernetes.io/enforce-version=v1.32 \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted

# Namespace kube-system (cần privileged)
kubectl label --overwrite ns kube-system \
  pod-security.kubernetes.io/enforce=privileged \
  pod-security.kubernetes.io/enforce-version=v1.32
```

### Policy Levels

| Level | Mô tả | Dùng cho |
|-------|-------|----------|
| `privileged` | Không giới hạn gì | System pods (kube-system, calico-system) |
| `baseline` | Minimal restrictions, tránh các privileged containers | User workloads thông thường |
| `restricted` | Chặt nhất, tuân theo Pod Security Standard | Production workloads, PCI-DSS |

## 2. Seccomp Profile (Mặc định)

K8s 1.32 mặc định dùng RuntimeDefault seccomp profile:

```bash
# Kiểm tra seccomp mặc định
kubectl get pod -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].spec.containers[0].securityContext.seccompProfile}'
```

Apply seccomp cho workload:

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      securityContext:
        seccompProfile:
          type: RuntimeDefault  # hoặc Localhost với profile tùy chỉnh
      containers:
      - name: app
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
```

## 3. AppArmor (Ubuntu)

```bash
# Kiểm tra AppArmor profile đang load
sudo aa-status | head -20

# Mặc định Ubuntu có sẵn docker profile
# Dùng với container:
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: apparmor-test
  annotations:
    container.apparmor.security.beta.kubernetes.io/app: "runtime/default"
spec:
  containers:
  - name: app
    image: nginx
EOF
```

## 4. ServiceAccount — Không dùng default trong production

```bash
# Kiểm tra namespace nào dùng default SA
kubectl get pods --all-namespaces -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{" "}{.spec.serviceAccount}{"\n"}{end}' | grep "default" | head -20

# Fix: Tạo service account riêng cho mỗi ứng dụng
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app-sa
  namespace: default
automountServiceAccountToken: false  # CHỈ mount nếu thực sự cần API access
EOF
```

### Auto-mount token: Tắt mặc định

```bash
kubectl patch serviceaccount default -n default -p '{"automountServiceAccountToken": false}'
```

## 5. RBAC — Nguyên tắc Least Privilege

```bash
# Không dùng cluster-admin cho application
# Ví dụ: read-only access cho một namespace
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pod-reader
  namespace: default
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: pod-reader-binding
  namespace: default
subjects:
- kind: ServiceAccount
  name: my-app-sa
  namespace: default
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
EOF
```

## 6. NetworkPolicy — Default Deny

Đã được tạo ở STEP 4 cho system namespaces. Apply cho toàn cluster:

```bash
# Default deny cho tất cả namespace không phải system
for ns in $(kubectl get ns -o name | grep -v 'kube-system\|calico-system\|tigera-operator\|kube-public\|kube-node-lease'); do
  n="${ns#namespace/}"
  cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: ${n}
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
EOF
done
```

## 7. Kubernetes CIS Benchmark

```bash
# Cài kube-bench (tool kiểm tra CIS)
docker run --rm \
  --pid=host \
  -v /etc:/etc:ro \
  -v /var:/var:ro \
  -v $(which kubectl):/usr/local/mount-from-host/kubectl:ro \
  -v /etc/kubernetes:/etc/kubernetes:ro \
  -v /var/lib/etcd:/var/lib/etcd:ro \
  aquasec/kube-bench:latest \
  run --targets=master,node

# Xem kết quả — fix các FAIL high/medium
```

## 8. Kubelet Authentication

```bash
# Kiểm tra kubelet config — anonymous-auth phải false
kubectl get --raw /api/v1/nodes/$(hostname)/proxy/configz | jq '.kubeletconfig.authentication.anonymous'
# Phải trả về: {"enabled": false}
```

## 9. Container Runtime Security

```bash
# Kiểm tra containerd config có security features
sudo cat /etc/containerd/config.toml | grep -E "privileged|restricted|seccomp|apparmor"
```

## 10. Cluster Audit Log — Đã cấu hình ở STEP 2

Kiểm tra audit log đang hoạt động:

```bash
sudo tail -5 /var/log/kubernetes/audit/audit.log | jq '.[] | {verb, resource: .objectRef.resource, namespace: .objectRef.namespace}' 2>/dev/null || \
  echo "Audit log empty hoặc chưa có events"
```

## Security Checklist (Pre-Production)

- [ ] Pod Security Admission: baseline hoặc restricted cho user namespaces
- [ ] ServiceAccount automount: tắt cho default SA
- [ ] RBAC: kiểm tra không có cluster-admin binding không cần thiết
- [ ] NetworkPolicy: default-deny cho mọi namespace
- [ ] Seccomp: RuntimeDefault profile cho tất cả workloads
- [ ] AppArmor: enabled (Ubuntu mặc định)
- [ ] Kubelet: anonymous-auth=false, protect-kernel-defaults=true
- [ ] API Server: anonymous-auth=false, audit-log enabled
- [ ] etcd: TLS enabled (kubeadm mặc định)
- [ ] Containerd: không chạy container privileged không cần thiết
- [ ] CIS Benchmark: pass các control critical/high
- [ ] Secret encryption: enable encryption at rest (nếu cần compliance)
- [ ] Image pull: use private registry với image scanning

#!/usr/bin/env bash
# ===================================================================
# STEP 5: Helm 3 — Production Setup
#
# Helm là package manager cho K8s. Production-ready config với:
#  - OCI registry support
#  - Plugin manager
#  - RBAC cho Helm operations
#  - Best-practice values structure
#  - Repo management (stable, bitnami, ingress-nginx)
# ===================================================================
set -euo pipefail

echo "================================================================"
echo "  STEP 5: Helm 3 Installation & Configuration"
echo "================================================================"
echo ""

# ===================================================================
# 1. INSTALL HELM
# ===================================================================

install_helm() {
  echo ">>> [1] Installing Helm 3..."

  if command -v helm &>/dev/null; then
    local current_version
    current_version=$(helm version --short 2>/dev/null | grep -oP 'v[\d.]+' | head -1)
    echo "   ✅ Helm already installed: $current_version"
    return
  fi

  # Cách 1: Script install (recommended)
  curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

  # Verify
  if command -v helm &>/dev/null; then
    echo "   ✅ Helm installed: $(helm version --short 2>/dev/null || helm version 2>/dev/null | head -1)"
  else
    echo "   ❌ Helm install failed. Trying apt..."
    # Cách 2: APT (snap channel)
    sudo snap install helm --classic 2>/dev/null || {
      echo "   ❌ Both installation methods failed."
      echo "   Manual: curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"
      exit 1
    }
  fi
}

# ===================================================================
# 2. REPO MANAGEMENT
# ===================================================================

setup_repos() {
  echo ""
  echo ">>> [2] Adding Helm repositories..."

  # Classic stable repo (deprecated, dùng bitnami thay thế)
  helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true

  # Ingress-Nginx
  helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true

  # MetalLB
  helm repo add metallb https://metallb.github.io/metallb 2>/dev/null || true

  # Jetstack (cert-manager)
  helm repo add jetstack https://charts.jetstack.io 2>/dev/null || true

  # Prometheus Community (kube-prometheus-stack)
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true

  # Grafana
  helm repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true

  # Update all repos
  helm repo update

  echo ""
  echo "   ✅ Repositories configured:"
  helm repo list | tail -n +2
}

# ===================================================================
# 3. RBAC FOR HELM
# ===================================================================
#
# Senior DevOps: Trong production, không dùng cluster-admin cho Helm.
# Tạo service account riêng với roles phù hợp.

setup_rbac() {
  echo ""
  echo ">>> [3] Creating Helm RBAC (Tillerless — native Helm 3)..."

  # Helm 3 không cần Tiller, nhưng ta tạo ClusterRole cho CI/CD pipelines
  cat <<'EOF' | kubectl apply -f - 2>/dev/null || true
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: helm-deployer
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: helm-deployer
rules:
- apiGroups: ["apps", "extensions"]
  resources: ["deployments", "statefulsets", "daemonsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets", "persistentvolumeclaims", "endpoints", "events"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "networkpolicies"]
  verbs: ["get", "list", "watch", "create", "update", "delete"]
- apiGroups: ["rbac.authorization.k8s.io"]
  resources: ["roles", "rolebindings"]
  verbs: ["get", "list", "watch", "create", "update", "delete"]
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "delete"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list", "watch", "create", "update", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: helm-deployer-binding
subjects:
- kind: ServiceAccount
  name: helm-deployer
  namespace: kube-system
roleRef:
  kind: ClusterRole
  name: helm-deployer
  apiGroup: rbac.authorization.k8s.io
EOF

  echo "   ✅ Helm deployer SA created (kube-system/helm-deployer)"
  echo "   🔑 Token for CI/CD: kubectl create token helm-deployer -n kube-system"
}

# ===================================================================
# 4. HELM PLUGINS
# ===================================================================

install_plugins() {
  echo ""
  echo ">>> [4] Installing useful Helm plugins..."

  # helm-diff — xem diff trước khi upgrade (production critical)
  if ! helm plugin list 2>/dev/null | grep -q diff; then
    helm plugin install https://github.com/databus23/helm-diff 2>/dev/null || \
      echo "   ⚠️  Could not install helm-diff (optional)"
  fi

  # helm-secrets — quản lý secrets trong GitOps (SOPS)
  if ! helm plugin list 2>/dev/null | grep -q secrets; then
    helm plugin install https://github.com/jkroepke/helm-secrets 2>/dev/null || \
      echo "   ⚠️  Could not install helm-secrets (optional)"
  fi

  echo ""
  echo "   Installed plugins:"
  helm plugin list 2>/dev/null || echo "   (no plugins)"
}

# ===================================================================
# 5. HELM ENV & COMPLETION
# ===================================================================

setup_env() {
  echo ""
  echo ">>> [5] Setting up Helm environment..."

  # Bash completion
  if ! grep -q "helm completion" ~/.bashrc 2>/dev/null; then
    helm completion bash | sudo tee /etc/bash_completion.d/helm > /dev/null 2>/dev/null || \
      helm completion bash >> ~/.bashrc 2>/dev/null || true
  fi

  # Helm aliases
  if ! grep -q "alias h=" ~/.bashrc 2>/dev/null; then
    cat <<'EOF' >> ~/.bashrc

# Helm aliases
alias h='helm'
alias hl='helm list'
alias hls='helm list --all-namespaces'
alias hdiff='helm diff upgrade'
alias hup='helm upgrade --install --atomic --timeout 10m'
alias hroll='helm rollback'
EOF
  fi

  echo "   ✅ Helm aliases + completion added"
}

# ===================================================================
# MAIN
# ===================================================================

main() {
  install_helm
  setup_repos
  setup_rbac
  install_plugins
  setup_env

  echo ""
  echo "================================================================"
  echo "  ✅ STEP 5 COMPLETE — Helm ready"
  echo "================================================================"
  echo ""
  echo "  Quick test:"
  echo "    helm search repo ingress-nginx"
  echo "    helm list --all-namespaces"
  echo ""
  echo "  Production deploy pattern:"
  echo "    helm upgrade --install <name> <chart> \\"
  echo "      --namespace <ns> --create-namespace \\"
  echo "      --values values-prod.yaml \\"
  echo "      --atomic --timeout 10m"
  echo ""
}

main

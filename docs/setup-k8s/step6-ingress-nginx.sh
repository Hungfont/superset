#!/usr/bin/env bash
# ===================================================================
# STEP 6: Ingress-Nginx Controller — Production Setup
#
# Senior DevOps decisions:
#  - Multiple replicas (HA) với PDB và HPA
#  - Host network: giảm latency, không cần kube-proxy overhead
#  - Proxy buffer tuning: tránh 502 cho large headers
#  - Metrics cho Prometheus
#  - Default SSL certificate (self-signed hoặc Let's Encrypt sau)
#  - Access log format chuẩn (JSON) cho log aggregation
#  - Custom error pages
#  - NodeSelector: chỉ chạy trên worker nodes
# ===================================================================
set -euo pipefail

# ===================================================================
# 0. CẤU HÌNH
# ===================================================================

NAMESPACE="ingress-nginx"

# High-availability
REPLICAS=2                           # Số replicas — tăng nếu cần

# Ingress class (mặc định)
INGRESS_CLASS="nginx"

# Mode
# "host-network"  → dùng hostPort: 80/443 (best performance, không cần MetalLB)
# "loadbalancer"  → dùng Service type LoadBalancer (cần MetalLB ở STEP 7)
# "nodeport"      → dùng NodePort (kém nhất, nhưng không cần MetalLB)
INGRESS_MODE="loadbalancer"

# Metrics cho Prometheus
ENABLE_METRICS=true

# SSL
GENERATE_DEFAULT_CERT=true           # Tạo self-signed cert mặc định
DEFAULT_CERT_DOMAIN="cluster.local"

# Resource requests/limits
REQUESTS_CPU="100m"
REQUESTS_MEM="256Mi"
LIMITS_CPU="500m"
LIMITS_MEM="512Mi"

# Proxy tuning
PROXY_BODY_SIZE="10m"                # Max body size (file upload)
PROXY_BUFFER_SIZE="16k"              # Header buffer
PROXY_BUFFERS="8 32k"               # 8 buffers, 32k each
PROXY_CONNECT_TIMEOUT="15s"
PROXY_READ_TIMEOUT="60s"
PROXY_SEND_TIMEOUT="60s"
KEEPALIVE_TIMEOUT="65s"

# Log format JSON cho log aggregation
ENABLE_JSON_LOG=true

# ===================================================================
# 1. CREATE NAMESPACE + DEFAULT SSL CERT
# ===================================================================

pre_setup() {
  echo ""
  echo "================================================================"
  echo "  PHASE 1: Prepare namespace & default SSL certificate"
  echo "================================================================"

  kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

  if [ "$GENERATE_DEFAULT_CERT" = true ]; then
    echo ">>> [1.1] Generating default self-signed SSL certificate..."

    # Tạo self-signed CA + cert cho ingress mặc định
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
      -keyout /tmp/default-ingress-key.pem \
      -out /tmp/default-ingress-cert.pem \
      -subj "/CN=${DEFAULT_CERT_DOMAIN}/O=Kubernetes" \
      -addext "subjectAltName=DNS:*.${DEFAULT_CERT_DOMAIN},DNS:*.local,DNS:localhost" 2>/dev/null

    kubectl create secret tls default-ssl-cert \
      --namespace "$NAMESPACE" \
      --cert=/tmp/default-ingress-cert.pem \
      --key=/tmp/default-ingress-key.pem \
      --dry-run=client -o yaml | kubectl apply -f -

    rm -f /tmp/default-ingress-*.pem
    echo "   ✅ Default SSL cert created: $NAMESPACE/default-ssl-cert"
  fi
}

# ===================================================================
# 2. GENERATE HELM VALUES
# ===================================================================

generate_values() {
  echo ""
  echo "================================================================"
  echo "  PHASE 2: Generate values-production.yaml"
  echo "================================================================"

  # Sinh đúng controller service type
  if [ "$INGRESS_MODE" = "host-network" ]; then
    SERVICE_TYPE=""
    HOST_NETWORK="true"
    KIND="DaemonSet"                # host-network bắt buộc DaemonSet
  elif [ "$INGRESS_MODE" = "loadbalancer" ]; then
    SERVICE_TYPE="LoadBalancer"
    HOST_NETWORK="false"
    KIND="Deployment"
  else
    SERVICE_TYPE="NodePort"
    HOST_NETWORK="false"
    KIND="Deployment"
  fi

  cat <<EOF | tee /tmp/ingress-nginx-values-prod.yaml > /dev/null
# ===================================================================
# Ingress-Nginx — Production Values
# Generated: $(date --iso-8601=seconds)
# ===================================================================

# --- Architecture ---
controller:
  kind: ${KIND}
  replicaCount: ${REPLICAS}
  minAvailable: 1                    # PDB: always keep 1 pod available
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 25%

  # --- Image ---
  image:
    registry: registry.k8s.io
    image: ingress-nginx/controller
    tag: "v1.12.0"                   # K8s 1.32 compatible
    digest: ""
    digestChroot: ""

  # --- Container ports ---
  containerPort:
    http: 80
    https: 443

  # --- Service ---
  service:
    enabled: true
    type: ${SERVICE_TYPE}
    annotations: {}
    # Senior DevOps: khi dùng AWS/GCP/Azure, thêm LB annotations ở đây
    # service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    # service.beta.kubernetes.io/azure-load-balancer-internal: "true"
    externalTrafficPolicy: Local     # Preserve source IP
    internal:
      enabled: false

  # --- Host Network ---
  hostNetwork: ${HOST_NETWORK}
  publishService:
    enabled: true
    pathOverride: ""

  # --- Ingress Class ---
  ingressClassResource:
    name: ${INGRESS_CLASS}
    enabled: true
    default: true
    controllerValue: "k8s.io/${INGRESS_CLASS}"
  ingressClass: ${INGRESS_CLASS}
  watchIngressWithoutClass: false

  # --- Resources ---
  resources:
    requests:
      cpu: "${REQUESTS_CPU}"
      memory: "${REQUESTS_MEM}"
    limits:
      cpu: "${LIMITS_CPU}"
      memory: "${LIMITS_MEM}"

  # --- Autoscaling (HPA) ---
  autoscaling:
    enabled: true
    minReplicas: ${REPLICAS}
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80
    behavior:
      scaleDown:
        stabilizationWindowSeconds: 300
        policies:
        - type: Pods
          value: 1
          periodSeconds: 60
      scaleUp:
        stabilizationWindowSeconds: 60
        policies:
        - type: Pods
          value: 2
          periodSeconds: 30

  # --- Topology Spread ---
  topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app.kubernetes.io/instance: ingress-nginx
        app.kubernetes.io/component: controller

  # --- Node Selection ---
  nodeSelector:
    topology.kubernetes.io/node-type: worker
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/control-plane
    operator: Exists

  # --- Metrics ---
  metrics:
    port: 10254
    enabled: ${ENABLE_METRICS}
    serviceMonitor:
      enabled: false                # Bật nếu có Prometheus Operator
      namespace: ${NAMESPACE}
      interval: 30s

  # --- Service Monitor ---
  # serviceMonitor:
  #   enabled: true
  #   namespace: monitoring

  # --- ConfigMap: Nginx tuning ---
  config:
    # === Performance ===
    worker-processes: "auto"
    worker-rlimit-nofile: "65536"
    max-worker-connections: "16384"
    keep-alive: "${KEEPALIVE_TIMEOUT}"
    keep-alive-requests: "10000"
    upstream-keepalive-connections: "320"
    upstream-keepalive-requests: "10000"
    upstream-keepalive-timeout: "60"

    # === Proxy ===
    proxy-body-size: "${PROXY_BODY_SIZE}"
    proxy-buffer-size: "${PROXY_BUFFER_SIZE}"
    proxy-buffers: "${PROXY_BUFFERS}"
    proxy-buffering: "on"
    proxy-connect-timeout: "${PROXY_CONNECT_TIMEOUT}"
    proxy-read-timeout: "${PROXY_READ_TIMEOUT}"
    proxy-send-timeout: "${PROXY_SEND_TIMEOUT}"
    proxy-next-upstream: "error timeout invalid_header http_502 http_503"
    proxy-next-upstream-tries: "3"
    proxy-protocol: "false"

    # === SSL/TLS ===
    ssl-protocols: "TLSv1.2 TLSv1.3"
    ssl-ciphers: "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384"
    ssl-ecdh-curve: "prime256v1:secp384r1:secp521r1"
    ssl-session-cache: "true"
    ssl-session-cache-size: "10m"
    ssl-session-timeout: "10m"
    ssl-prefer-server-ciphers: "true"
    ssl-redirect: "true"

    # === Headers & Security ===
    hide-headers: "Server,X-Powered-By"
    server-tokens: "false"
    hsts: "true"
    hsts-max-age: "31536000"
    hsts-include-subdomains: "true"
    hsts-preload: "true"
    use-forwarded-headers: "true"
    compute-full-forwarded-for: "true"
    forwarded-for-header: "X-Forwarded-For"
    enable-real-ip: "true"
    forwarded-for-fallback: "127.0.0.1"
    real-ip-header: "X-Forwarded-For"
    real-ip-recursive: "true"
    set-real-ip-from: "0.0.0.0/0"

    # === Timeouts ===
    client-header-timeout: "60"
    client-body-timeout: "60"
    client-max-body-size: "${PROXY_BODY_SIZE}"
    large-client-header-buffers: "4 16k"

    # === Logging ===
    # Senior DevOps: JSON format cho log aggregation (Loki, ES, etc.)
    log-format-escape-json: "${ENABLE_JSON_LOG}"
    log-format-upstream: '{"time": "$$time_iso8601", "remote_addr": "$$remote_addr", "x-forward-for": "$$proxy_add_x_forwarded_for", "request_id": "$$req_id", "remote_user": "$$remote_user", "bytes_sent": $$bytes_sent, "request_time": $$request_time, "status": $$status, "vhost": "$$host", "request_proto": "$$server_protocol", "path": "$$uri", "request_query": "$$args", "request_length": $$request_length, "duration": $$request_time, "method": "$$request_method", "http_referrer": "$$http_referer", "http_user_agent": "$$http_user_agent" }'
    access-log-path: "/var/log/nginx/access.log"
    error-log-path: "/var/log/nginx/error.log"
    error-log-level: "notice"

    # === Rate Limiting (DDoS protection) ===
    limit-rate-after: "10m"
    limit-rate: "50k"

  # --- Custom error pages ---
  # Senior DevOps: Tạo ConfigMap với error pages nếu cần
  custom-http-errors: |-
    404
    503
    502

  # --- Default SSL certificate ---
  extraArgs:
    default-ssl-certificate: "${NAMESPACE}/default-ssl-cert"

# --- Pod Disruption Budget ---
podDisruptionBudget:
  enabled: true
  maxUnavailable: 1

# --- Default Backend (404 page) ---
defaultBackend:
  enabled: true
  name: defaultbackend
  image:
    registry: registry.k8s.io
    image: defaultbackend-amd64
    tag: "1.5"
  resources:
    requests:
      cpu: 10m
      memory: 20Mi
    limits:
      cpu: 50m
      memory: 64Mi
  nodeSelector:
    topology.kubernetes.io/node-type: worker

# --- ServiceAccount ---
serviceAccount:
  create: true
  name: ingress-nginx-controller
  automountServiceAccountToken: true
EOF

  echo "   ✅ Values file: /tmp/ingress-nginx-values-prod.yaml"
}

# ===================================================================
# 3. DEPLOY INGRESS-NGINX
# ===================================================================

deploy_ingress() {
  echo ""
  echo "================================================================"
  echo "  PHASE 3: Deploy Ingress-Nginx"
  echo "================================================================"

  # Add repo & update (đã có từ STEP 5, nhưng đảm bảo)
  helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
  helm repo update ingress-nginx 2>/dev/null || true

  echo ""
  echo ">>> [3.1] Installing ingress-nginx via Helm..."
  helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
    --namespace "$NAMESPACE" --create-namespace \
    --values /tmp/ingress-nginx-values-prod.yaml \
    --atomic \
    --timeout 10m \
    --wait \
    --debug 2>&1 | tail -20

  echo ""
  echo ">>> [3.2] Waiting for controller to be ready..."
  kubectl -n "$NAMESPACE" rollout status deployment/ingress-nginx-controller --timeout=120s 2>/dev/null || \
    kubectl -n "$NAMESPACE" rollout status daemonset/ingress-nginx-controller --timeout=120s 2>/dev/null || true

  echo ""
  echo ">>> [3.3] Pods:"
  kubectl -n "$NAMESPACE" get pods -o wide
}

# ===================================================================
# 4. VERIFY
# ===================================================================

verify() {
  echo ""
  echo "================================================================"
  echo "  PHASE 4: Verification"
  echo "================================================================"

  echo ">>> [4.1] IngressClass:"
  kubectl get ingressclass "$INGRESS_CLASS" -o yaml 2>/dev/null | head -10

  echo ""
  echo ">>> [4.2] Service:"
  kubectl -n "$NAMESPACE" get svc

  echo ""
  echo ">>> [4.3] Check admission webhook:"
  kubectl -n "$NAMESPACE" get validatingwebhookconfiguration -l app.kubernetes.io/instance=ingress-nginx 2>/dev/null | head -5 || \
    echo "   (webhook may take a moment)"

  echo ""
  if [ "$INGRESS_MODE" = "loadbalancer" ]; then
    local EXTERNAL_IP
    EXTERNAL_IP=$(kubectl -n "$NAMESPACE" get svc ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    echo ">>> LoadBalancer IP: $EXTERNAL_IP"
    if [ "$EXTERNAL_IP" = "pending" ]; then
      echo "   ℹ️  Cần MetalLB (STEP 7) để cấp IP. Tiếp tục..."
    fi
  fi
}

# ===================================================================
# MAIN
# ===================================================================

main() {
  pre_setup
  generate_values
  deploy_ingress
  verify

  echo ""
  echo "================================================================"
  echo "  ✅ STEP 6 COMPLETE — Ingress-Nginx deployed"
  echo "================================================================"
  echo ""
  echo "📌 Quick test (sau khi có MetalLB):"
  echo "    kubectl create deploy test --image=nginx --port=80"
  echo "    kubectl expose deploy test --port=80"
  echo "    kubectl create ingress test --class=nginx --rule=\"test.local/*=test:80\""
  echo ""
  echo "📌 Check logs:"
  echo "    kubectl -n ${NAMESPACE} logs -l app.kubernetes.io/component=controller"
  echo ""
}

main

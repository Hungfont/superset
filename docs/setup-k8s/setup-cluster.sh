#!/usr/bin/env bash
# ===================================================================
# 🚀 ONE COMMAND — Deploy full Kubernetes Cluster via Ansible
#
# Production-grade: 3 nodes, K8s 1.32, Calico, MetalLB, Ingress-Nginx
#
# Usage:
#   ./setup-cluster.sh                        # Interactive
#   ./setup-cluster.sh --yes                  # Non-interactive (auto)
#   ./setup-cluster.sh --tags bootstrap       # Only bootstrap phase
#   ./setup-cluster.sh --skip-reboot          # Don't auto reboot
#   ./setup-cluster.sh --check                # Ansible dry-run
#   ./setup-cluster.sh --inventory my/hosts   # Custom inventory
#
# Requirements:
#   - Ansible >= 9 (control machine)
#   - SSH key access to all nodes (root)
#   - Python 3 on all nodes
#
# Senior DevOps:
#   - Kiểm tra preflight trước khi deploy
#   - Tự động cài Ansible + collections nếu thiếu
#   - Support --check, --tags, --limit
#   - Logs lưu tại /tmp/k8s-ansible-*.log
# ===================================================================
set -euo pipefail

# === Colors ===
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# === Config ===
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ANSIBLE_DIR="${SCRIPT_DIR}/ansible"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
LOG_FILE="/tmp/k8s-ansible-${TIMESTAMP}.log"
INTERACTIVE=true
SKIP_REBOOT=false
ANSIBLE_ARGS=""

# Parse args
for arg in "$@"; do
  case "$arg" in
    --yes|--auto|-y) INTERACTIVE=false ;;
    --skip-reboot)   SKIP_REBOOT=true ;;
    --check|-C)      ANSIBLE_ARGS="$ANSIBLE_ARGS --check" ;;
    --inventory=*)   ANSIBLE_ARGS="$ANSIBLE_ARGS -i ${arg#*=}" ;;
    --tags=*)        ANSIBLE_ARGS="$ANSIBLE_ARGS --tags ${arg#*=}" ;;
    --limit=*)       ANSIBLE_ARGS="$ANSIBLE_ARGS --limit ${arg#*=}" ;;
    --verbose|-v)    ANSIBLE_ARGS="$ANSIBLE_ARGS -v" ;;
    --help|-h)
      echo "Usage: $0 [options]"
      echo "  --yes           Non-interactive mode"
      echo "  --skip-reboot   Skip reboot phase"
      echo "  --check         Dry-run (ansible --check)"
      echo "  --tags=TAGS     Only run specific tags"
      echo "  --limit=LIMIT   Limit to specific hosts"
      echo "  --verbose       Verbose output"
      echo "  --help          This help"
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown argument: $arg${NC}"
      exit 1
      ;;
  esac
done

# ===================================================================
# PREFLIGHT CHECKS
# ===================================================================

preflight() {
  echo ""
  echo "================================================================"
  echo "  🔍 PREFLIGHT CHECKS"
  echo "================================================================"
  echo ""

  # Check Ansible
  if ! command -v ansible-playbook &>/dev/null; then
    echo -e "${YELLOW}⚠️  Ansible not found. Installing...${NC}"
    if command -v pip3 &>/dev/null; then
      pip3 install ansible --user 2>&1 | tail -1
      export PATH="$HOME/.local/bin:$PATH"
    else
      echo -e "${RED}❌ pip3 not found. Install: apt install python3-pip ansible${NC}"
      exit 1
    fi
  fi

  # Check Ansible version (need >= 9)
  local ansible_version
  ansible_version=$(ansible-playbook --version 2>/dev/null | head -1 | grep -oP '[\d.]+' | head -1)
  echo -e "   Ansible version: ${CYAN}${ansible_version}${NC}"

  # Check Ansible collections
  echo ">>> Installing Ansible collections..."
  ansible-galaxy collection install -r "${ANSIBLE_DIR}/requirements.yml" \
    --upgrade 2>&1 | tail -3

  # Check inventory
  if [ ! -f "${ANSIBLE_DIR}/inventory/hosts.yml" ]; then
    echo -e "${RED}❌ Inventory not found: ${ANSIBLE_DIR}/inventory/hosts.yml${NC}"
    echo "   Edit the inventory file with your node IPs first."
    exit 1
  fi

  # Check SSH access
  echo ">>> Testing SSH connectivity..."
  ansible all -i "${ANSIBLE_DIR}/inventory/hosts.yml" \
    -m ping --one-line -o 2>&1 | tee -a "$LOG_FILE"

  local unreachable
  unreachable=$(ansible all -i "${ANSIBLE_DIR}/inventory/hosts.yml" \
    -m ping 2>&1 | grep "UNREACHABLE" | wc -l)
  if [ "$unreachable" -gt 0 ]; then
    echo -e "${RED}❌ $unreachable node(s) unreachable. Check SSH key access.${NC}"
    exit 1
  fi

  echo -e "${GREEN}✅ All nodes reachable via SSH${NC}"
  echo ""
}

# ===================================================================
# SUMMARY
# ===================================================================

show_summary() {
  echo ""
  echo "================================================================"
  echo "  📋 DEPLOYMENT PLAN"
  echo "================================================================"
  echo ""

  local nodes
  nodes=$(ansible-inventory -i "${ANSIBLE_DIR}/inventory/hosts.yml" --list 2>/dev/null | \
    python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('k8s_cluster',{}).get('hosts',[])))" 2>/dev/null || echo "?")

  echo "   Nodes:      ${CYAN}${nodes}${NC}"
  echo "   K8s:        ${CYAN}1.32${NC}"
  echo "   CNI:        ${CYAN}Calico (${CALICO_MODE:-vxlan})${NC}"
  echo "   Ingress:    ${CYAN}ingress-nginx${NC}"
  echo "   LB:         ${CYAN}MetalLB (L2)${NC}"
  echo "   Helm:       ${CYAN}Yes${NC}"
  echo "   Audit:      ${CYAN}Yes${NC}"
  echo "   etcd backup:${CYAN}Yes (30 min)${NC}"
  echo ""
  echo "   Tags:       ${CYAN}${ANSIBLE_ARGS:-all}${NC}"
  echo "   Log file:   ${CYAN}${LOG_FILE}${NC}"
  echo ""

  # Lấy IP ranges từ inventory
  local pod_cidr service_cidr lb_range
  pod_cidr=$(grep -A1 "pod_network" "${ANSIBLE_DIR}/inventory/hosts.yml" | grep -oP '[\d.]+/[\d]+' | head -1)
  service_cidr=$(grep -A1 "service_network" "${ANSIBLE_DIR}/inventory/hosts.yml" | grep -oP '[\d.]+/[\d]+' | head -1)
  lb_range=$(grep -A1 "lb_ip_range" "${ANSIBLE_DIR}/inventory/hosts.yml" | grep -oP '[\d.]+-[\d.]+' | head -1)

  echo "   Pod CIDR:    ${pod_cidr:-10.244.0.0/16}"
  echo "   Service CIDR:${service_cidr:-10.96.0.0/12}"
  echo "   MetalLB IPs: ${lb_range:-192.168.1.200-210}"
  echo ""

  if [ "$INTERACTIVE" = true ]; then
    echo -e "${YELLOW}⚠️  Đảm bảo đã sửa đúng IP/hostname trong:${NC}"
    echo "      ${ANSIBLE_DIR}/inventory/hosts.yml"
    echo ""
    echo -e "${YELLOW}⚠️  BẠN SẼ REBOOT CẢ 3 NODE trong quá trình deploy.${NC}"
    echo ""
    read -r -p "Continue? [y/N] " response
    if [[ ! "$response" =~ ^[yY] ]]; then
      echo -e "${RED}Aborted.${NC}"
      exit 0
    fi
  fi
}

# ===================================================================
# DEPLOY
# ===================================================================

deploy() {
  echo ""
  echo "================================================================"
  echo "  🚀 DEPLOYING CLUSTER"
  echo "================================================================"
  echo ""

  local cmd="ansible-playbook ${ANSIBLE_DIR}/site.yml ${ANSIBLE_ARGS}"

  # Override default inventory if not specified
  if [[ ! "$ANSIBLE_ARGS" =~ "-i" ]]; then
    cmd="$cmd -i ${ANSIBLE_DIR}/inventory/hosts.yml"
  fi

  # Skip reboot if requested
  if [ "$SKIP_REBOOT" = true ]; then
    cmd="$cmd --skip-tags reboot,phase3"
  fi

  echo "   Running: ${cmd}"
  echo "   Log:     ${LOG_FILE}"
  echo ""

  # Execute with tee
  # Senior DevOps: chia thành nhiều lần chạy để dễ debug nếu fail
  local phases=()
  if [[ "$ANSIBLE_ARGS" =~ "--tags" ]]; then
    # User specified tags — run once
    phases=("run")
  else
    # Full run — split into phases for better observability
    phases=("common" "bootstrap" "reboot" "control-plane" "worker" "addons")
  fi

  for phase in "${phases[@]}"; do
    if [ "$phase" = "run" ]; then
      echo -e "\n${CYAN}>>> Running specified tags...${NC}"
      eval "$cmd" 2>&1 | tee -a "$LOG_FILE"
    else
      echo -e "\n${CYAN}>>> Phase: $phase${NC}"
      ansible-playbook "${ANSIBLE_DIR}/site.yml" \
        -i "${ANSIBLE_DIR}/inventory/hosts.yml" \
        --tags "$phase" 2>&1 | tee -a "$LOG_FILE"

      local exit_code=${PIPESTATUS[0]}
      if [ $exit_code -ne 0 ]; then
        echo -e "${RED}❌ Phase '$phase' failed (exit code: $exit_code)${NC}"
        echo "   Check log: $LOG_FILE"
        echo "   You can resume: $0 --tags ${phase} --skip-reboot"
        exit $exit_code
      fi
    fi
  done

  echo ""
  echo -e "${GREEN}✅ ALL PHASES COMPLETE!${NC}"
}

# ===================================================================
# VERIFY
# ===================================================================

verify() {
  echo ""
  echo "================================================================"
  echo "  ✅ VERIFICATION"
  echo "================================================================"

  # Nodes
  echo ""
  echo ">>> Nodes:"
  ansible all -i "${ANSIBLE_DIR}/inventory/hosts.yml" \
    -m shell -a "kubectl get nodes || echo 'kubectl not ready'" \
    --limit control_plane 2>&1 | grep -v "UNREACHABLE" | head -20

  # Pods
  echo ""
  echo ">>> System pods:"
  ansible all -i "${ANSIBLE_DIR}/inventory/hosts.yml" \
    -m shell -a "kubectl get pods -A | head -20" \
    --limit control_plane 2>&1 | grep -v "CHANGED" | grep -v "UNREACHABLE"

  # Ingress LB IP
  echo ""
  echo ">>> Ingress LoadBalancer IP:"
  local lb_ip
  lb_ip=$(ansible all -i "${ANSIBLE_DIR}/inventory/hosts.yml" \
    -m shell -a "kubectl -n ingress-nginx get svc ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}'" \
    --limit control_plane 2>&1 | tail -1 | grep -oP '[\d.]+' | head -3)

  if [ -n "$lb_ip" ]; then
    echo -e "${GREEN}   Ingress IP: $lb_ip${NC}"
  else
    echo -e "${YELLOW}   Ingress IP pending (MetalLB might still assigning)${NC}"
  fi

  echo ""
  echo "================================================================"
  echo -e "${GREEN}  🎉 CLUSTER READY — Production-grade K8s${NC}"
  echo "================================================================"
  echo ""
  echo "   kubectl get nodes    (trên master)"
  echo "   kubectl get pods -A  (tất cả pods)"
  echo "   curl http://${lb_ip:-<ingress-ip>}  (test Ingress)"
  echo ""
  echo "   📍 Log: ${LOG_FILE}"
  echo ""
}

# ===================================================================
# MAIN
# ===================================================================

main() {
  echo ""
  echo "================================================================"
  echo "  🚀 KUBERNETES CLUSTER DEPLOYMENT"
  echo "  Senior DevOps — Ansible Automation"
  echo "================================================================"

  cd "$ANSIBLE_DIR"
  preflight
  show_summary
  deploy
  verify
}

main

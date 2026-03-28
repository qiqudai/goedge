#!/usr/bin/env bash
set -euo pipefail

APP_NAME="goedge-node-tune"

APPLY=0
SKIP_SYSCTL=0
SKIP_LIMITS=0

BANDWIDTH_MBPS=""
RTT_MS=50

usage() {
  cat <<'EOF'
用法: optimize-node.sh [options]

选项:
  --apply                    实际写入配置并应用
  --bandwidth-mbps <num>     设置带宽(Mbps)，用于计算 TCP 缓冲区
  --rtt-ms <num>             RTT(ms)，默认 50
  --skip-sysctl              跳过 sysctl 配置
  --skip-limits              跳过 limits 配置
  -h|--help                  显示帮助

示例:
  ./optimize-node.sh --bandwidth-mbps 10000 --apply
  ./optimize-node.sh --bandwidth-mbps 2000 --rtt-ms 20 --apply
EOF
}

log() { printf "%s\n" "$*"; }
warn() { printf "WARN: %s\n" "$*" >&2; }
die() { printf "ERROR: %s\n" "$*" >&2; exit 1; }

require_root() {
  if [[ $EUID -ne 0 ]]; then
    die "需要 root 权限执行 --apply。"
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --apply) APPLY=1; shift ;;
      --bandwidth-mbps) BANDWIDTH_MBPS="${2:-}"; shift 2 ;;
      --rtt-ms) RTT_MS="${2:-}"; shift 2 ;;
      --skip-sysctl) SKIP_SYSCTL=1; shift ;;
      --skip-limits) SKIP_LIMITS=1; shift ;;
      -h|--help) usage; exit 0 ;;
      *) die "未知参数: $1" ;;
    esac
  done
}

detect_os() {
  if [[ -r /etc/os-release ]]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    OS_ID="${ID:-unknown}"
    OS_VERSION="${VERSION_ID:-unknown}"
  else
    OS_ID="unknown"
    OS_VERSION="unknown"
  fi
}

mem_gb() {
  local kb
  kb=$(awk '/MemTotal/ {print $2}' /proc/meminfo)
  printf "%d" $(( (kb + 1048575) / 1048576 ))
}

cpu_cores() {
  nproc
}

detect_bandwidth() {
  if [[ -n "$BANDWIDTH_MBPS" ]]; then
    return
  fi
  if command -v ethtool >/dev/null 2>&1; then
    local iface speed
    iface=$(ip -o link show up | awk -F': ' '$2 != "lo" {print $2; exit}')
    if [[ -n "$iface" ]]; then
      speed=$(ethtool "$iface" 2>/dev/null | awk -F'[: ]+' '/Speed/ {print $3}')
      if [[ -n "$speed" && "$speed" != "Unknown!" ]]; then
        BANDWIDTH_MBPS="$speed"
      fi
    fi
  fi
  if [[ -z "$BANDWIDTH_MBPS" ]]; then
    BANDWIDTH_MBPS=1000
    warn "未检测到网卡速率，默认使用 ${BANDWIDTH_MBPS} Mbps。可用 --bandwidth-mbps 指定。"
  fi
}

calc_sysctl_values() {
  local bw="$BANDWIDTH_MBPS"
  local rtt="$RTT_MS"
  local bdp buf
  bdp=$(awk -v bw="$bw" -v rtt="$rtt" 'BEGIN{print int(bw*1000000/8*rtt/1000)}')
  buf=$(awk -v b="$bdp" 'BEGIN{
    v=b*2; if(v<16777216) v=16777216; if(v>268435456) v=268435456; print int(v)
  }')
  SYSCTL_RMEM_MAX="$buf"
  SYSCTL_WMEM_MAX="$buf"
  SYSCTL_TCP_RMEM="4096 87380 ${buf}"
  SYSCTL_TCP_WMEM="4096 65536 ${buf}"
}

calc_limits() {
  local mem
  mem=$(mem_gb)
  MAX_FD=$(( mem * 65536 ))
  if [[ "$MAX_FD" -lt 131072 ]]; then MAX_FD=131072; fi
  if [[ "$MAX_FD" -gt 1048576 ]]; then MAX_FD=1048576; fi
}

plan_summary() {
  log "系统: ${OS_ID} ${OS_VERSION}"
  log "内存: $(mem_gb) GB"
  log "CPU: $(cpu_cores) cores"
  log "带宽: ${BANDWIDTH_MBPS} Mbps, RTT: ${RTT_MS} ms"
  log "MAX_FD: ${MAX_FD}"
}

write_sysctl() {
  local sysctl_file="/etc/sysctl.d/99-goedge-tuning.conf"
  cat >"$sysctl_file" <<EOF
# ${APP_NAME}
fs.file-max = ${MAX_FD}
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 16384
net.core.rmem_max = ${SYSCTL_RMEM_MAX}
net.core.wmem_max = ${SYSCTL_WMEM_MAX}
net.ipv4.ip_local_port_range = 10240 65535
net.ipv4.tcp_max_syn_backlog = 16384
net.ipv4.tcp_fin_timeout = 15
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_mtu_probing = 1
net.ipv4.tcp_rmem = ${SYSCTL_TCP_RMEM}
net.ipv4.tcp_wmem = ${SYSCTL_TCP_WMEM}
net.core.default_qdisc = fq
EOF

  if sysctl net.ipv4.tcp_available_congestion_control 2>/dev/null | grep -q bbr; then
    printf "net.ipv4.tcp_congestion_control = bbr\n" >>"$sysctl_file"
  else
    warn "bbr 不可用，跳过 tcp_congestion_control 设置。"
  fi

  sysctl -p "$sysctl_file"
}

write_limits() {
  local limits_file="/etc/security/limits.d/99-goedge.conf"
  cat >"$limits_file" <<EOF
# ${APP_NAME}
root soft nofile ${MAX_FD}
root hard nofile ${MAX_FD}
EOF
}

apply_all() {
  require_root

  if [[ "$SKIP_SYSCTL" -eq 0 ]]; then
    write_sysctl
  fi
  if [[ "$SKIP_LIMITS" -eq 0 ]]; then
    write_limits
  fi
}

main() {
  parse_args "$@"
  detect_os
  detect_bandwidth
  calc_limits
  calc_sysctl_values

  plan_summary

  if [[ "$APPLY" -eq 1 ]]; then
    apply_all
    log "完成。"
  else
    log "未执行 --apply，仅输出建议。"
  fi
}

main "$@"

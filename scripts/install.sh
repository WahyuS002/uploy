#!/usr/bin/env bash
#
# scripts/install.sh — One-time server bootstrap for Uploy managed proxy.
#
# Usage:
#   sudo bash scripts/install.sh
#
# This script prepares a server so that the Uploy backend can deploy
# applications with FQDN-based routing via Traefik. It is meant to be
# run once per server by an operator with root access.

set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
readonly TARGET_USER="uploy"
readonly PROXY_DIR="/data/uploy/proxy"
DOCKER_DNS=""
SKIP_DNS_CONFIG=false
CHECK_ONLY=false
FORCE=false

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
readonly COLOR_GREEN="\033[0;32m"
readonly COLOR_RED="\033[0;31m"
readonly COLOR_YELLOW="\033[0;33m"
readonly COLOR_RESET="\033[0m"

pass() { printf "  ${COLOR_GREEN}PASS${COLOR_RESET}  %s\n" "$1"; }
fail() { printf "  ${COLOR_RED}FAIL${COLOR_RESET}  %s\n" "$1"; }
warn() { printf "  ${COLOR_YELLOW}WARN${COLOR_RESET}  %s\n" "$1"; }
info() { printf "  ....  %s\n" "$1"; }

die() {
    printf "${COLOR_RED}ERROR:${COLOR_RESET} %s\n" "$1" >&2
    exit 1
}

# ---------------------------------------------------------------------------
# OS detection
# ---------------------------------------------------------------------------
detect_os() {
    if [[ -f /etc/os-release ]]; then
        # shellcheck source=/dev/null
        . /etc/os-release
        OS_TYPE="${ID,,}" # lowercase
    else
        die "Cannot detect OS: /etc/os-release not found."
    fi
}

# ---------------------------------------------------------------------------
# Docker installation per distro
# ---------------------------------------------------------------------------
install_docker() {
    info "Installing Docker for '$OS_TYPE'. This may take a while..."

    case "$OS_TYPE" in
        ubuntu|debian)
            apt-get update -y >/dev/null 2>&1
            apt-get install -y ca-certificates curl gnupg >/dev/null 2>&1
            install -m 0755 -d /etc/apt/keyrings
            curl -fsSL "https://download.docker.com/linux/$OS_TYPE/gpg" \
                | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null
            chmod a+r /etc/apt/keyrings/docker.gpg
            echo \
                "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
                https://download.docker.com/linux/$OS_TYPE $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
                | tee /etc/apt/sources.list.d/docker.list >/dev/null
            apt-get update -y >/dev/null 2>&1
            apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null 2>&1
            ;;
        centos|rhel|rocky|almalinux)
            dnf install -y dnf-plugins-core >/dev/null 2>&1 || true
            if command -v dnf5 &>/dev/null; then
                dnf config-manager addrepo --from-repofile="https://download.docker.com/linux/centos/docker-ce.repo" --overwrite >/dev/null 2>&1
            else
                dnf config-manager --add-repo="https://download.docker.com/linux/centos/docker-ce.repo" >/dev/null 2>&1
            fi
            dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null 2>&1
            ;;
        fedora)
            dnf install -y dnf-plugins-core >/dev/null 2>&1 || true
            if command -v dnf5 &>/dev/null; then
                dnf config-manager addrepo --from-repofile="https://download.docker.com/linux/fedora/docker-ce.repo" --overwrite >/dev/null 2>&1
            else
                dnf config-manager --add-repo="https://download.docker.com/linux/fedora/docker-ce.repo" >/dev/null 2>&1
            fi
            dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null 2>&1
            ;;
        amzn)
            dnf install -y docker >/dev/null 2>&1
            # Amazon Linux ships docker without compose plugin — install manually
            DOCKER_CONFIG=${DOCKER_CONFIG:-/usr/local/lib/docker}
            mkdir -p "$DOCKER_CONFIG/cli-plugins"
            curl -fsSL "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
                -o "$DOCKER_CONFIG/cli-plugins/docker-compose"
            chmod +x "$DOCKER_CONFIG/cli-plugins/docker-compose"
            ;;
        *)
            die "Unsupported distro '$OS_TYPE'. Please install Docker manually: https://docs.docker.com/engine/install/"
            ;;
    esac

    # Enable and start Docker
    if command -v systemctl &>/dev/null; then
        systemctl enable docker >/dev/null 2>&1
        systemctl start docker >/dev/null 2>&1
    fi

    # Verify installation succeeded
    if ! command -v docker &>/dev/null; then
        die "Docker installation failed. Please install manually: https://docs.docker.com/engine/install/"
    fi
}

usage() {
    cat <<'EOF'
Usage: sudo bash scripts/install.sh [OPTIONS]

Optional flags:
  --docker-dns <csv>        Upstream DNS servers for Docker containers (e.g. 1.1.1.1,8.8.8.8)
  --skip-dns-config         Skip Docker daemon DNS configuration
  --check-only              Audit prerequisites without modifying the system
  --force                   Allow overwrite/backup of existing configuration
  -h, --help                Show this help message
EOF
    exit 0
}

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --docker-dns)   DOCKER_DNS="$2"; shift 2 ;;
        --skip-dns-config) SKIP_DNS_CONFIG=true; shift ;;
        --check-only)   CHECK_ONLY=true; shift ;;
        --force)        FORCE=true; shift ;;
        -h|--help)      usage ;;
        *) die "Unknown flag: $1. Use --help for usage." ;;
    esac
done


# ---------------------------------------------------------------------------
# Track overall result
# ---------------------------------------------------------------------------
OVERALL_OK=true
mark_fail() { OVERALL_OK=false; }

# ---------------------------------------------------------------------------
# 1. Preflight
# ---------------------------------------------------------------------------
echo ""
echo "=== Preflight ==="

# Must be root
if [[ "$(id -u)" -ne 0 ]]; then
    die "This script must be run as root (use sudo)."
fi
pass "Running as root"

# Must be Linux
if [[ "$(uname -s)" != "Linux" ]]; then
    die "This script only supports Linux hosts. Detected: $(uname -s)"
fi
pass "Host is Linux"

# Detect distro
detect_os
pass "Detected OS: $OS_TYPE"

# Docker — install automatically if missing
if command -v docker &>/dev/null; then
    pass "Docker already installed"
else
    if [[ "$CHECK_ONLY" == true ]]; then
        fail "Docker is not installed"
        mark_fail
    else
        install_docker
        pass "Docker installed successfully"
    fi
fi

# Docker Compose
if docker compose version &>/dev/null; then
    pass "Docker Compose available ($(docker compose version --short 2>/dev/null || echo 'unknown'))"
else
    if [[ "$CHECK_ONLY" == true ]]; then
        fail "Docker Compose plugin is not available"
        mark_fail
    else
        die "Docker Compose plugin is not available after installation. Please install docker-compose-plugin manually."
    fi
fi

# Ensure platform user exists
if id "$TARGET_USER" &>/dev/null; then
    pass "User '$TARGET_USER' already exists"
else
    if [[ "$CHECK_ONLY" == true ]]; then
        fail "User '$TARGET_USER' does not exist"
        mark_fail
    else
        useradd -m -s /bin/bash "$TARGET_USER"
        pass "Created platform user '$TARGET_USER'"
    fi
fi

# ---------------------------------------------------------------------------
# 2. Docker Access Bootstrap
# ---------------------------------------------------------------------------
echo ""
echo "=== Docker Access Bootstrap ==="

# Docker docs: the daemon binds to a root-owned Unix socket by default.
# To allow a non-root deploy user to run `docker` without sudo, Docker's
# Linux post-install guide recommends the `docker` group + `usermod -aG`,
# followed by a fresh login so group membership is re-evaluated:
# https://docs.docker.com/engine/install/linux-postinstall/
#
# Ensure docker group exists
if ! getent group docker &>/dev/null; then
    if [[ "$CHECK_ONLY" == true ]]; then
        fail "Group 'docker' does not exist"
        mark_fail
    else
        groupadd docker
        pass "Created group 'docker'"
    fi
else
    pass "Group 'docker' exists"
fi

# Add user to docker group
if id -nG "$TARGET_USER" 2>/dev/null | grep -qw docker; then
    pass "User '$TARGET_USER' is already in group 'docker'"
else
    if [[ "$CHECK_ONLY" == true ]]; then
        fail "User '$TARGET_USER' is not in group 'docker'"
        mark_fail
    else
        usermod -aG docker "$TARGET_USER"
        pass "Added '$TARGET_USER' to group 'docker'"
        warn "User '$TARGET_USER' may need to log out and back in for group membership to take effect"
    fi
fi

# ---------------------------------------------------------------------------
# 3. Filesystem Bootstrap
# ---------------------------------------------------------------------------
echo ""
echo "=== Filesystem Bootstrap ==="

if [[ -d "$PROXY_DIR" ]]; then
    pass "Proxy directory '$PROXY_DIR' already exists"
else
    if [[ "$CHECK_ONLY" == true ]]; then
        fail "Proxy directory '$PROXY_DIR' does not exist"
        mark_fail
    else
        mkdir -p "$PROXY_DIR"
        pass "Created proxy directory '$PROXY_DIR'"
    fi
fi

# Set ownership so the target user can write docker-compose.yaml, acme.json, etc.
if [[ "$CHECK_ONLY" == false ]] && [[ -d "$PROXY_DIR" ]]; then
    chown -R "$TARGET_USER":"$TARGET_USER" "$PROXY_DIR"
    pass "Ownership of '$PROXY_DIR' set to '$TARGET_USER'"
else
    # In check-only mode, verify ownership
    if [[ -d "$PROXY_DIR" ]]; then
        DIR_OWNER=$(stat -c '%U' "$PROXY_DIR" 2>/dev/null || echo "unknown")
        if [[ "$DIR_OWNER" == "$TARGET_USER" ]]; then
            pass "Proxy directory owned by '$TARGET_USER'"
        else
            fail "Proxy directory owned by '$DIR_OWNER', expected '$TARGET_USER'"
            mark_fail
        fi
    fi
fi

# ---------------------------------------------------------------------------
# 4. Docker DNS Bootstrap
# ---------------------------------------------------------------------------
echo ""
echo "=== Docker DNS Bootstrap ==="

# Docker docs note two behaviors that matter here:
# 1. Containers on custom networks use Docker's embedded DNS server.
# 2. Docker also warns that `--dns=127.0.0.1` points at the container's own
#    loopback, not the host's. In other words, loopback addresses in the
#    127.0.0.0/8 range refer back to the current network namespace itself.
#
# Why this matters for Uploy:
# - Many Linux hosts use a local stub resolver on the host, such as
#   `systemd-resolved` on 127.0.0.53.
# - Docker's troubleshooting docs explicitly call out loopback resolver
#   configs such as 127.0.0.1 and 127.0.1.1 as not working inside containers.
# - 127.0.0.53 is the same class of problem: a host-local loopback resolver.
#   If Docker forwards container DNS queries to that host stub, ACME lookups
#   from Traefik can fail even though the host itself resolves names fine.
#
# This bootstrap therefore writes daemon-level DNS config only when the host
# appears to use a local stub resolver, matching Docker's documented
# troubleshooting guidance and daemon DNS options:
# https://docs.docker.com/engine/network/
# https://docs.docker.com/engine/daemon/troubleshoot/
# https://docs.docker.com/reference/cli/dockerd/
#
DAEMON_JSON="/etc/docker/daemon.json"
DNS_CHANGED=false

if [[ "$SKIP_DNS_CONFIG" == true ]]; then
    info "Skipped (--skip-dns-config)"
else
    # Determine DNS servers to configure
    if [[ -n "$DOCKER_DNS" ]]; then
        # Convert comma-separated to JSON array: "1.1.1.1,8.8.8.8" -> ["1.1.1.1","8.8.8.8"]
        DNS_JSON_ARRAY=$(echo "$DOCKER_DNS" | tr ',' '\n' | sed 's/^.*$/    "&"/' | paste -sd ',' - | sed 's/^/[\n/;s/$/\n  ]/' )
        # Build a cleaner JSON array
        DNS_JSON_ARRAY="["
        FIRST=true
        IFS=',' read -ra DNS_SERVERS <<< "$DOCKER_DNS"
        for server in "${DNS_SERVERS[@]}"; do
            server=$(echo "$server" | xargs) # trim whitespace
            if [[ "$FIRST" == true ]]; then
                DNS_JSON_ARRAY+="\"$server\""
                FIRST=false
            else
                DNS_JSON_ARRAY+=", \"$server\""
            fi
        done
        DNS_JSON_ARRAY+="]"
    else
        # Default: check if host uses a local resolver (for example
        # systemd-resolved on 127.0.0.53). Docker documents loopback stub
        # resolvers such as 127.0.0.1 and 127.0.1.1 as problematic inside
        # containers because they resolve to the container namespace itself.
        NEEDS_DNS=false
        if [[ -f /etc/resolv.conf ]]; then
            if grep -Eq '127\.0\.0\.53|127\.0\.0\.1|127\.0\.1\.1' /etc/resolv.conf 2>/dev/null; then
                NEEDS_DNS=true
            fi
        fi

        if [[ "$NEEDS_DNS" == true ]]; then
            DNS_JSON_ARRAY='["1.1.1.1", "8.8.8.8"]'
            info "Host uses local resolver — will configure Docker DNS explicitly"
        else
            pass "Host DNS resolver looks fine for containers"
            DNS_JSON_ARRAY=""
        fi
    fi

    if [[ -n "$DNS_JSON_ARRAY" ]]; then
        DESIRED_DNS="$DNS_JSON_ARRAY"

        if [[ -f "$DAEMON_JSON" ]]; then
            # File exists — check if we need to modify it
            EXISTING_DNS=$(python3 -c "
import json, sys
try:
    with open('$DAEMON_JSON') as f:
        data = json.load(f)
    print(json.dumps(data.get('dns', [])))
except Exception:
    print('[]')
" 2>/dev/null || echo "[]")

            DESIRED_DNS_NORMALIZED=$(python3 -c "
import json
print(json.dumps(json.loads('$DESIRED_DNS')))
" 2>/dev/null || echo "$DESIRED_DNS")

            if [[ "$EXISTING_DNS" == "$DESIRED_DNS_NORMALIZED" ]]; then
                pass "Docker DNS already configured correctly"
            else
                # daemon.json has other content — need to merge
                if [[ "$CHECK_ONLY" == true ]]; then
                    fail "Docker DNS needs updating in $DAEMON_JSON"
                    mark_fail
                else
                    # Check if safe to merge
                    CAN_MERGE=$(python3 -c "
import json, sys
try:
    with open('$DAEMON_JSON') as f:
        data = json.load(f)
    # We can safely merge if it's a dict
    if not isinstance(data, dict):
        print('no')
    else:
        print('yes')
except json.JSONDecodeError:
    print('no')
except Exception:
    print('no')
" 2>/dev/null || echo "no")

                    if [[ "$CAN_MERGE" == "no" && "$FORCE" == false ]]; then
                        fail "Cannot safely merge into $DAEMON_JSON (invalid JSON or unexpected format). Use --force to overwrite with backup."
                        mark_fail
                    else
                        # Backup existing file
                        BACKUP="${DAEMON_JSON}.bak.$(date +%Y%m%d%H%M%S)"
                        cp "$DAEMON_JSON" "$BACKUP"
                        info "Backed up $DAEMON_JSON to $BACKUP"

                        if [[ "$CAN_MERGE" == "yes" ]]; then
                            python3 -c "
import json
with open('$DAEMON_JSON') as f:
    data = json.load(f)
data['dns'] = json.loads('$DESIRED_DNS')
with open('$DAEMON_JSON', 'w') as f:
    json.dump(data, f, indent=2)
    f.write('\n')
"
                        else
                            # Force mode: overwrite with minimal config
                            printf '{\n  "dns": %s\n}\n' "$DESIRED_DNS" > "$DAEMON_JSON"
                        fi
                        DNS_CHANGED=true
                        pass "Updated Docker DNS configuration"
                    fi
                fi
            fi
        else
            # No daemon.json exists — create it
            if [[ "$CHECK_ONLY" == true ]]; then
                fail "Docker DNS not configured ($DAEMON_JSON does not exist)"
                mark_fail
            else
                mkdir -p /etc/docker
                printf '{\n  "dns": %s\n}\n' "$DESIRED_DNS" > "$DAEMON_JSON"
                DNS_CHANGED=true
                pass "Created $DAEMON_JSON with DNS configuration"
            fi
        fi
    fi

    # Restart Docker only if config changed
    if [[ "$DNS_CHANGED" == true ]]; then
        info "Restarting Docker daemon to apply DNS changes..."
        systemctl restart docker
        pass "Docker daemon restarted"
    fi
fi

# ---------------------------------------------------------------------------
# 5. Final Verification
# ---------------------------------------------------------------------------
echo ""
echo "=== Final Verification ==="

RESULTS=()

# Docker available
if command -v docker &>/dev/null && docker info &>/dev/null; then
    pass "Docker is available"
    RESULTS+=("Docker: OK")
else
    fail "Docker is not available"
    RESULTS+=("Docker: FAIL")
    mark_fail
fi

# Docker Compose available
if docker compose version &>/dev/null; then
    pass "Docker Compose is available"
    RESULTS+=("Docker Compose: OK")
else
    fail "Docker Compose is not available"
    RESULTS+=("Docker Compose: FAIL")
    mark_fail
fi

# User in docker group
if id -nG "$TARGET_USER" 2>/dev/null | grep -qw docker; then
    pass "User '$TARGET_USER' in docker group"
    RESULTS+=("User '$TARGET_USER' in docker group: OK")
else
    fail "User '$TARGET_USER' not in docker group"
    RESULTS+=("User '$TARGET_USER' in docker group: FAIL")
    mark_fail
fi

# Proxy dir exists and writable by target user
if [[ -d "$PROXY_DIR" ]]; then
    if su -s /bin/sh "$TARGET_USER" -c "test -w '$PROXY_DIR'" 2>/dev/null; then
        pass "Proxy dir '$PROXY_DIR' exists and writable"
        RESULTS+=("Proxy dir '$PROXY_DIR': OK")
    else
        fail "Proxy dir '$PROXY_DIR' exists but not writable by '$TARGET_USER'"
        RESULTS+=("Proxy dir '$PROXY_DIR': FAIL (not writable)")
        mark_fail
    fi
else
    fail "Proxy dir '$PROXY_DIR' does not exist"
    RESULTS+=("Proxy dir '$PROXY_DIR': FAIL")
    mark_fail
fi

# Docker DNS from inside container
# Final verification uses a short containerized DNS lookup because Docker's
# troubleshooting guidance recommends validating name resolution from inside a
# container, not only from the host.
if [[ "$SKIP_DNS_CONFIG" == false ]]; then
    if docker run --rm alpine:3.20 nslookup example.com &>/dev/null; then
        pass "Docker container DNS resolution works"
        RESULTS+=("Docker DNS: OK")
    else
        fail "Docker container DNS resolution failed"
        RESULTS+=("Docker DNS: FAIL")
        mark_fail
    fi
else
    info "Docker DNS check skipped (--skip-dns-config)"
    RESULTS+=("Docker DNS: SKIPPED")
fi


# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "==========================================="
echo " Summary"
echo "==========================================="
for r in "${RESULTS[@]}"; do
    echo "  $r"
done
echo ""

if [[ "$OVERALL_OK" == true ]]; then
    printf "  ${COLOR_GREEN}Server ready for Uploy managed proxy: YES${COLOR_RESET}\n"
else
    printf "  ${COLOR_RED}Server ready for Uploy managed proxy: NO${COLOR_RESET}\n"
fi

# Manual follow-up reminders
echo ""
echo "Manual follow-up (if not already done):"
echo "  - Log out and back in as '$TARGET_USER' for docker group membership to take effect"
echo "  - Open inbound ports 80 and 443 in your firewall / security group"
echo "  - Point your domain DNS records to this server's public IP"
echo ""

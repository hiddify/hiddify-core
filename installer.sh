#!/bin/bash

# Hiddify Core Universal Installer
# Supports OpenWrt, Ubuntu, Debian, CentOS, and other Linux distributions.

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Hiddify Core Universal Installer${NC}"

# Check for root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root${NC}"
    exit 1
fi

# Detect OS
OS="linux"
if [ -f /etc/openwrt_release ]; then
    OS="openwrt"
    echo -e "Detected OS: ${GREEN}OpenWrt${NC}"
elif [ -f /etc/os-release ]; then
    . /etc/os-release
    OS_NAME=$ID
    echo -e "Detected OS: ${GREEN}$OS_NAME${NC}"
else
    echo -e "Detected OS: ${GREEN}Generic Linux${NC}"
fi

# Detect Architecture
ARCH_RAW=$(uname -m)
case "$ARCH_RAW" in
    x86_64) ARCH="amd64" ;;
    i386|i686) ARCH="386" ;;
    aarch64) ARCH="arm64" ;;
    armv7*) ARCH="armv7" ;;
    armv6*) ARCH="armv6" ;;
    armv5*) ARCH="armv5" ;;
    mips64el) ARCH="mips64le" ;;
    mips64) ARCH="mips64" ;;
    mipsel) ARCH="mipsle" ;;
    mips) ARCH="mips" ;;
    riscv64) ARCH="riscv64" ;;
    ppc64le) ARCH="ppc64le" ;;
    s390x) ARCH="s390x" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH_RAW${NC}"; exit 1 ;;
esac

# Check for libc type (glibc vs musl) for Linux
LIBC=""
if [ "$OS" != "openwrt" ]; then
    if ldd --version 2>&1 | grep -iq musl; then
        LIBC="-musl"
    elif ldd --version 2>&1 | grep -iq glibc; then
        LIBC="-glibc"
    fi
fi

# Determine artifact name
# Note: For OpenWrt, we usually prefer generic or musl. 
# For standard Linux, we use the specific libc variant if available, otherwise generic.
ARTIFACT="hiddify-core-linux-${ARCH}${LIBC}.tar.gz"

# Fetch latest version
echo -e "Fetching latest version information..."
REPO="hiddify/hiddify-core"
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo -e "${RED}Failed to fetch latest version tag. Attempting fallback...${NC}"
    LATEST_TAG="v4.0.4" # Fallback
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$ARTIFACT"

echo -e "Downloading ${BLUE}$ARTIFACT${NC} ($LATEST_TAG)..."
WORKDIR=$(mktemp -d)
curl -L "$DOWNLOAD_URL" -o "$WORKDIR/$ARTIFACT" || {
    # If specific libc variant fails, try generic
    if [ -n "$LIBC" ]; then
        echo -e "Libc-specific artifact not found, trying generic..."
        ARTIFACT="hiddify-core-linux-${ARCH}.tar.gz"
        DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$ARTIFACT"
        curl -L "$DOWNLOAD_URL" -o "$WORKDIR/$ARTIFACT"
    fi
}

# Install Binary
echo -e "Installing binary to /usr/bin/hiddify-core..."
tar -zxf "$WORKDIR/$ARTIFACT" -C "$WORKDIR"
# Find the binary in the tarball (it might be in a subdir or renamed)
BIN_PATH=$(find "$WORKDIR" -type f -name "hiddify*" -executable | head -n 1)
mv "$BIN_PATH" /usr/bin/hiddify-core
chmod +x /usr/bin/hiddify-core

# Setup Config
mkdir -p /etc/hiddify-core
if [ ! -f /etc/hiddify-core/config.json ]; then
    echo -e "Creating default configuration..."
    cat <<EOF > /etc/hiddify-core/config.json
{
  "log": { "level": "info" },
  "dns": { "servers": [{ "address": "tls://8.8.8.8" }] },
  "inbounds": [{ "type": "shadowsocks", "listen": "::", "listen_port": 8080, "network": "tcp", "method": "2022-blake3-aes-128-gcm", "password": "Gn1JUS14bLUHgv1cWDDp4A==", "multiplex": { "enabled": true, "padding": true } }],
  "outbounds": [{ "type": "direct" }, { "type": "dns", "tag": "dns-out" }],
  "route": { "rules": [{ "port": 53, "outbound": "dns-out" }] }
}
EOF
fi

# Service Configuration
if [ "$OS" = "openwrt" ]; then
    echo -e "Configuring ${GREEN}OpenWrt procd${NC} service..."
    
    # UCI Config
    if [ ! -f /etc/config/hiddify-core ]; then
        cat <<EOF > /etc/config/hiddify-core
config hiddify-core 'main'
	option enabled '1'
	option conffile '/etc/hiddify-core/config.json'
	option workdir '/usr/share/hiddify-core'
	option log_stderr '1'
EOF
    fi

    # Init Script
    cat <<'EOF' > /etc/init.d/hiddify-core
#!/bin/sh /etc/rc.common
USE_PROCD=1
START=99
PROG="/usr/bin/hiddify-core"
start_service() {
  config_load "hiddify-core"
  local enabled conffile workdir log_stderr
  config_get_bool enabled "main" "enabled" "0"
  [ "$enabled" -eq "1" ] || return 0
  config_get conffile "main" "conffile" "/etc/hiddify-core/config.json"
  config_get workdir "main" "workdir" "/usr/share/hiddify-core"
  config_get_bool log_stderr "main" "log_stderr" "1"
  mkdir -p "$workdir"
  procd_open_instance
  procd_set_param command "$PROG" run -c "$conffile" -D "$workdir"
  procd_set_param file "$conffile"
  procd_set_param stderr "$log_stderr"
  procd_set_param respawn
  procd_close_instance
}
EOF
    chmod +x /etc/init.d/hiddify-core
    /etc/init.d/hiddify-core enable
    /etc/init.d/hiddify-core restart

else
    echo -e "Configuring ${GREEN}systemd${NC} service..."
    
    # Systemd Service
    cat <<EOF > /etc/systemd/system/hiddify-core.service
[Unit]
Description=hiddify-core service
After=network.target nss-lookup.target network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/hiddify-core -D /var/lib/hiddify-core run -c /etc/hiddify-core/config.json
Restart=on-failure
RestartSec=10s
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
EOF

    mkdir -p /var/lib/hiddify-core
    systemctl daemon-reload
    systemctl enable hiddify-core
    systemctl restart hiddify-core
fi

echo -e "${GREEN}Installation successful!${NC}"
echo -e "Binary: /usr/bin/hiddify-core"
echo -e "Config: /etc/hiddify-core/config.json"
if [ "$OS" = "openwrt" ]; then
    echo -e "Service: /etc/init.d/hiddify-core"
else
    echo -e "Service: systemctl status hiddify-core"
fi

rm -rf "$WORKDIR"

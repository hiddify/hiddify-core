#!/bin/bash

# Function to install dialog
install_dialog() {
    case "$OSTYPE" in
        linux-gnu*)
            if [ -f /etc/debian_version ]; then
                sudo apt-get update && sudo apt-get install dialog -y  # Debian/Ubuntu
            elif [ -f /etc/redhat-release ]; then
                sudo yum install dialog -y  # CentOS/RHEL
            elif [ -f /etc/fedora-release ]; then
                sudo dnf install dialog -y  # Fedora
            elif [ -f /etc/arch-release ]; then
                sudo pacman -S dialog  # Arch Linux
            else
                echo "Unsupported Linux distribution."
                exit 1
            fi
            ;;
        darwin*)
            brew install dialog  # macOS
            ;;
        openwrt*)
            opkg update
            opkg install dialog  # OpenWrt
            ;;
        *)
            echo "Installation method for your OS is not supported."
            exit 1
            ;;
    esac
}

# Function to detect architecture
detect_architecture() {
    ARCH=$(uname -m)

    case $ARCH in
        x86_64)
            ARCH_NAME="amd64"
            ;;
        i686 | i386)
            ARCH_NAME="386"
            ;;
        armv7l | arm)
            ARCH_NAME="armv7"
            ;;
        aarch64)
            ARCH_NAME="arm64"
            ;;
        mips)
            ARCH_NAME="mips-hardfloat"
            ;;
        mipsel)
            ARCH_NAME="mipsel-hardfloat"
            ;;
        *)
            dialog --msgbox "Unknown architecture: $ARCH\nPlease manually download the appropriate version." 10 40
            exit 1
            ;;
    esac

    dialog --msgbox "Detected architecture: $ARCH_NAME" 10 30
}

# Function to download the appropriate file
download_file() {
    RELEASE_VERSION="latest"
    DOWNLOAD_URL="https://github.com/hiddify/hiddify-core/releases/download/${RELEASE_VERSION}/hiddify-cli-linux-${ARCH_NAME}.tar.gz"

    dialog --infobox "Downloading HiddifyCli for architecture $ARCH_NAME from $DOWNLOAD_URL..." 10 40
    wget -O /tmp/HiddifyCli.tar.gz "$DOWNLOAD_URL"

    if [ $? -ne 0 ]; then
        dialog --msgbox "Error downloading the file! Please check your internet connection." 10 40
        exit 1
    fi
}

# Main script execution
detect_architecture
download_file

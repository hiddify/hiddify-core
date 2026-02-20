<p align="center">
  <img src="https://raw.githubusercontent.com/hiddify/hiddify.com/refs/heads/main/docs/assets/hiddify-app-logo.svg" alt="Hiddify Logo" width="128">
</p>

<h1 align="center">Hiddify Core</h1>

<p align="center">
  <strong>The Ultimate Universal Proxy Platform</strong><br>
  A powerful, high-performance core for the Hiddify ecosystem, supporting all major protocols and platforms.
</p>

<p align="center">
  <a href="https://hiddify.com"><img src="https://img.shields.io/badge/Website-hiddify.com-blue?style=flat-square" alt="Website"></a>
  <a href="https://t.me/hiddify"><img src="https://img.shields.io/badge/Telegram-Join-blue?style=flat-square&logo=telegram" alt="Telegram"></a>
  <img src="https://img.shields.io/github/license/hiddify/hiddify-core?style=flat-square" alt="License">
  <img src="https://img.shields.io/github/v/release/hiddify/hiddify-core?style=flat-square" alt="Version">
</p>

---

## 🚀 Quick Setup

Install `hiddify-core` on any Linux platform (Ubuntu, Debian, CentOS, OpenWrt, and more) with a single command:

```bash
bash <(curl https://i.hiddify.com/core)
```
or 
```bash
bash <(curl -Ls https://raw.githubusercontent.com/hiddify/hiddify-core/main/installer.sh)
```

> [!NOTE]
> This script automatically detects your OS and architecture, installs the appropriate binary, and configures the service manager (Systemd or Procd).

---

## ✨ Key Features

- **🌐 Multi-Protocol Support**: Naive, Mieru, Hysteria, SOCKS, Shadowsocks, ShadowTLS, Tor, Trojan, VLess, VMess, WireGuard, and more.
- **📱 Cross-Platform**: Powering Hiddify on Android, macOS, Linux, Windows, and iOS.
- **🔌 Extension System**: Powerful third-party extension capability to modify configs and add custom features.
- **⚡ High Performance**: Optimized core built on top of `sing-box` for maximum speed and stability.
- **🏠 Router Ready**: Native support for OpenWrt and other router platforms.

---

## 🛠 Installation Methods

### 🐳 Docker
Quickly deploy as a containerized service:

```bash
# Pull image
docker pull ghcr.io/hiddify/hiddify-core:latest

# Or using Docker Compose
git clone https://github.com/hiddify/hiddify-core
cd hiddify-core/docker
docker-compose up -d
```

### 📶 OpenWrt
For manual installation or advanced configuration on OpenWrt, refer to our [OpenWrt Setup Guide](platform/wrt/README.md).

---

## Extension

An extension is something that can be added to hiddify application by a third party. It will add capability to modify configs, do some extra action, show and receive data from users.

This extension will be shown in all Hiddify Platforms such as Android/macOS/Linux/Windows/iOS

[Create an extension](https://github.com/hiddify/hiddify-app-example-extension)

Features and Road map:

- [x] Add Third Party Extension capability
- [x] Test Extension from Browser without any dependency to android/mac/.... `./cmd.sh extension` the open browser `https://127.0.0.1:12346`
- [x] Show Custom UI from Extension `github.com/hiddify/hiddify-core/extension.UpdateUI()` 
- [x] Show Custom Dialog from Extension `github.com/hiddify/hiddify-core/extension.ShowDialog()`
- [x] Show Alert Dialog from Extension `github.com/hiddify/hiddify-core/extension.ShowMessage()` 
- [x] Get Data from UI `github.com/hiddify/hiddify-core/extension.SubmitData()` 
- [x] Save Extension Data from `e.Base.Data`
- [x] Load Extension Data to `e.Base.Data`
- [x] Disable / Enable Extension 
- [x] Update user proxies before connecting `github.com/hiddify/hiddify-core/extension.BeforeAppConnect()` 
- [x] Run Tiny Independent Instance  `github.com/hiddify/hiddify-core/extension/sdk.RunInstance()` 
- [x] Parse Any type of configs/url  `github.com/hiddify/hiddify-core/extension/sdk.ParseConfig()` 
- [ ] ToDo: Add Support for MultiLanguage Interface
- [ ] ToDo: Custom Extension Outbound
- [ ] ToDo: Custom Extension Inbound
- [ ] ToDo: Custom Extension ProxyConfig
 
 Demo Screenshots from HTML:
 
 <img width="531" alt="image" src="https://github.com/user-attachments/assets/0fbef76f-896f-4c45-a6b8-7a2687c47013">
 <img width="531" alt="image" src="https://github.com/user-attachments/assets/15bccfa0-d03e-4354-9368-241836d82948">


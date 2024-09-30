# hiddify-core


## Docker
To Run our docker image see https://github.com/hiddify/hiddify-core/pkgs/container/hiddify-core

Docker
```
docker pull ghcr.io/hiddify/hiddify-core:latest
```

Docker Compose
```
git clone https://github.com/hiddify/hiddify-core
cd hiddify-core/docker
docker-compose up
```

## OpenWrt
### Configure and enable Hiddify service
```ash
root@OpenWrt:~# uci set hiddify.main.config='https://www.example.com/sub.txt'
root@OpenWrt:~# uci set hiddify.main.enabled='1'
root@OpenWrt:~# uci commit hiddify
root@OpenWrt:~# service hiddify restart
```
### Forward all traffic to the tun interface by assigning it to firewall wan zone.

Command Line: 

```ash
root@OpenWrt:~# uci add_list firewall.@zone[1].device='tun+'
root@OpenWrt:~# uci commit firewall
root@OpenWrt:~# service firewall restart
```
Luci : 

Select Firewall from the Network tab and edit the wan zone.
Add tun+ under covered devices on the Advanced Settings tab,
Save and Apply .

![image](https://github.com/user-attachments/assets/ef431c51-9f58-4bf5-afe6-4b1eca0feca7)


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


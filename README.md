# hiddify-core

## Docker

To Run our docker image see <https://github.com/hiddify/hiddify-core/pkgs/container/hiddify-core>

Docker

```bash
docker pull ghcr.io/hiddify/hiddify-core:latest
```

Docker Compose

```bash
git clone https://github.com/hiddify/hiddify-core
cd hiddify-core/docker
docker-compose up
```

## WRT

...

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
- [x] Parse Any type of configs/url `github.com/hiddify/hiddify-core/extension/sdk.ParseConfig()`
- [ ] ToDo: Add Support for MultiLanguage Interface
- [ ] ToDo: Custom Extension Outbound
- [ ] ToDo: Custom Extension ProxyConfig

Demo Screenshots from HTML:

![image](https://github.com/user-attachments/assets/0fbef76f-896f-4c45-a6b8-7a2687c47013)
![image](https://github.com/user-attachments/assets/15bccfa0-d03e-4354-9368-241836d82948)

## SingBox

Hiddify-core به صورت پیش‌فرض بر پایه `sing-box` نسخه `v1.13.0-alpha.20` (مطابق `go.mod`) اجرا می‌شود و از ساختار رسمی معرفی‌شده در [مستندات sing-box](https://sing-box.sagernet.org/configuration/) پیروی می‌کند.

### مفاهیم پایه

- **ساختار پیکربندی**

  ```jsonc
  {
    "log": {},
    "dns": {},
    "ntp": {},
    "certificate": {},
    "endpoints": [],
    "inbounds": [],
    "outbounds": [],
    "route": {},
    "services": [],
    "experimental": {}
  }
  ```

  این اسکلت با سند `index.md` در مستندات رسمی هم‌راستاست و توسط توابع `config.BuildConfig()` و `config.BuildConfigJson()` ایجاد و تکمیل می‌شود.
- **راستی‌آزمایی و قالب‌بندی**

  - اجرای `sing-box check` برای اعتبارسنجی نهایی.
  - اجرای `sing-box format -w -c config.json` جهت یکسان‌سازی قالب.
  - در صورت نیاز به ادغام چند فایل: `sing-box merge output.json -c config.json`.

- **منابع رسمی برای فیلدها**

  - ورودی‌ها (Inbound): [Inbound](https://sing-box.sagernet.org/configuration/inbound/)
  - خروجی‌ها (Outbound): [Outbound](https://sing-box.sagernet.org/configuration/outbound/)
  - مسیریابی (Route): [Route](https://sing-box.sagernet.org/configuration/route/)

### گردش کار در هسته

- **تجمیع گزینه‌ها**: ساختار `config.HiddifyOptions` تنظیمات برنامه را نگهداری می‌کند و با `option.Options` کتابخانه sing-box ادغام می‌شود.
- **تولید خودکار کانفیگ**:
  - تابع `config.BuildConfig()` ماژول‌های DNS، Inbound، Outbound، و Rules را با آخرین استانداردهای sing-box تنظیم می‌کند.
  - تابع `config.ParseConfigContent()` ورودی‌های Clash، V2Ray یا JSON را به ساختار sing-box تبدیل و با `libbox.CheckConfig` اعتبارسنجی می‌کند.
- **به‌روزرسانی Warp و گزینه‌های پویا**: ماژول‌های `config/warp.go` و `config/outbound.go` پروفایل‌های WireGuard/WARP را مطابق API‌های sing-box 1.13 نگاشت می‌کنند.

### نکات نسخه‌ای و مهاجرت

- **توجه به تغییرات 1.11 تا 1.13**: قوانین DNS و Route در نسخه‌های اخیر به Rule-Setها منتقل شده‌اند. برای مهاجرت کانفیگ‌های قدیمی از راهنمای [Migration](https://sing-box.sagernet.org/migration/) استفاده کنید.
- **فعال‌سازی ویژگی‌های جدید**: گزینه‌های `Route.AutoDetectInterface` و `Experimental.ClashAPI` در `config/config.go` مطابق توصیه‌های رسمی فعال شده‌اند.
- **پیگیری انتشارها**: به‌منظور همگام‌سازی با تغییرات آینده، به مخزن [SagerNet/sing-box](https://github.com/SagerNet/sing-box) و صفحه انتشار نسخه‌ها مراجعه کنید.

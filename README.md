# 🥋 Keyboard Sensei (macOS)

**Keyboard Sensei**, İngilizce (ANSI) fiziksel klavyelerde Türkçe Q klavye düzeni kullanılırken yaşanan eksik tuş (`<`, `>`, `|` vb.) sorununu çözmek için tasarlanmış ultra hafif, yerel bir macOS tuş dönüştürücüsüdür.

Artık bir betik değil; **tam bir macOS Uygulaması (`Keyboard Sensei.app`)** ve bilgisayarınız açıldığında arka planda sessizce çalışan **macOS Başlangıç Servisi (LaunchAgent)** olarak yapılandırılmıştır.

---

## 🌟 Neler Yapıldı?

1. **Yerel macOS Uygulaması (`Keyboard Sensei.app`):**
   * `/Applications/Keyboard Sensei.app` içine kuruldu.
   * `LSUIElement=true` sayesinde Dock'ta kalabalık yapmaz, arka plan servisi gibi sessizce çalışır.
   * Özel tasarlanmış macOS uygulama ikonuna (`AppIcon.icns`) sahiptir.

2. **Başlangıçta Otomatik Çalışma (macOS LaunchAgent):**
   * Bilgisayarınızı yeniden başlattığınızda veya oturum açtığınızda **arka planda otomatik başlar**.
   * Web paneli üzerinden (`http://localhost:5252`) **"🚀 Otomatik Başlat"** anahtarı ile tek tıkla açılıp kapatılabilir.

3. **Modern Web Yönetim Paneli (`http://localhost:5252`):**
   * **İnteraktif Tuş Kaydedici:** İstediğiniz tuşa basarak kısayolu otomatik yakalayın.
   * **Canlı Test Alanı (Sandbox):** Kısayollarınızı hemen test edin.
   * **Canlı Olay Akışı:** Tuş dönüşümlerini anlık izleyin.
   * **Tek Tıkla ANSI Şablonu:** Eksik karakterleri (`<`, `>`, `|`, `~`, `` ` ``) tek tıkla yükler.

---

## 🔐 İlk Kurulum: macOS Erişilebilirlik İzni

macOS'in klavye tuşlarını yakalayabilmesi için uygulamanın bir kereliğe mahsus onaylanması gerekir:

1. **Sistem Ayarları** $\rightarrow$ **Gizlilik ve Güvenlik** $\rightarrow$ **Erişilebilirlik** bölümüne gidin.
2. Listede **`Keyboard Sensei`** uygulamasını bulun ve anahtarını **Açık** yapın (Eğer listede yoksa alttaki `+` butonuna basıp `/Applications/Keyboard Sensei.app` dosyasını seçin).
3. İzni verdiğiniz anda arka plan servisi otomatik olarak tuşları yakalamaya başlar (yeniden başlatma gerekmez).

---

## 🕹️ CLI Komutları (İsteğe Bağlı)

```bash
# Servisi başlangıca kur (Zaten kuruldu)
/Applications/Keyboard\ Sensei.app/Contents/MacOS/keyboard-sensei --install

# Servisi başlangıçtan kaldır
/Applications/Keyboard\ Sensei.app/Contents/MacOS/keyboard-sensei --uninstall

# Servis durumunu kontrol et
/Applications/Keyboard\ Sensei.app/Contents/MacOS/keyboard-sensei --service-status

# Web panelini tarayıcıda aç
open http://localhost:5252
```

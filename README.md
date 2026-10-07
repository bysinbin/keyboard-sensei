# 🥋 Keyboard Sensei (macOS)

**Keyboard Sensei**, İngilizce (ANSI) fiziksel klavyelerde Türkçe Q klavye düzeni kullanılırken yaşanan eksik tuş (`<`, `>`, `|` vb.) sorununu çözmek için tasarlanmış ultra hafif, yerel bir macOS tuş dönüştürücüsü ve makro yönetim aracıdır.

Artık bir betik değil; **tam bir macOS Uygulaması (`Keyboard Sensei.app`)**, menü çubuğunda çalışan **Status Bar (Tray)** aracı ve bilgisayarınız açıldığında arka planda sessizce çalışan **macOS Başlangıç Servisi (LaunchAgent)** olarak yapılandırılmıştır.

---

## 🌟 Öne Çıkan Özellikler

1. **🍏 macOS Menü Çubuğu (Status Bar Item):**
   * Menü çubuğunda (sağ üst saat yanı) **🥋 Sensei ikonu**.
   * Tek tıkla *Duraklat / Başlat*, *Web Panelini Aç*, *Açılışta Otomatik Başlatma* kontrolü ve *Çıkış*.
   * Motorun aktiflik durumunu anlık gösterir.

2. **🎹 Görsel İnteraktif ANSI Klavye Haritası:**
   * Web panelinde fiziksel ANSI klavye şeması üzerinde hangi tuşların hangi kombinasyonlarla haritalandığını neon ışıklarla gösterir.
   * Klavyedeki herhangi bir tuşa tıklayarak o tuşa anında yeni bir kural ekleyebilirsiniz.

3. **📂 Çoklu Profil Yönetimi & Hazır Şablonlar:**
   * **ANSI ➔ Türkçe Q (Varsayılan):** `<` (⌘ö), `>` (⌘ç), `|` (⌥.), `~` (⌥-), `` ` `` (⌥\).
   * **Geliştirici / Kodlama:** `<<`, `>>`, `=>` ve `{date}` tarih damgası kısayolları.
   * **Uzak Masaüstü (RDP / Windows):** Windows sanal makineleri için özel karakter haritalaması.
   * **JSON İçe / Dışa Aktar:** Profillerinizi ve ayarlarınızı tek tıkla yedekleyin veya paylaşın.

4. **🎯 Uygulamaya Özel Kısıtlamalar (App Whitelist & Blacklist):**
   * İstediğiniz kısayolu yalnızca belirli uygulamalarda aktif yapabilir (örn: `Code`, `Terminal`) veya belirli uygulamalarda devre dışı bırakabilirsiniz (örn: `Figma`, `Slack`).
   * "Odaktaki Uygulamayı Algıla" butonuyla anlık çalışan uygulamanın adı ve Bundle ID'si otomatik doldurulur.

5. **📝 Dinamik Metin Genişletme & Makrolar (Snippets):**
   * Kısayollar sadece tek tuş değil; çok karakterli metinleri ve dinamik değişkenleri yazabilir:
     * `{date}` $\rightarrow$ Günün tarihi (`2026-10-07`)
     * `{time}` $\rightarrow$ Anlık saat (`21:58:30`)
     * `{datetime}` $\rightarrow$ Tarih ve saat
     * `{uuid}` $\rightarrow$ Benzersiz rastgele ID

6. **⌨️ Donanım / Klavye Cihazı Yönetimi (Hardware Device Filtering):**
   * macOS IOKit HID seviyesinde Mac'inize bağlı tüm dahili (MacBook klavyesi) ve harici (USB / Bluetooth) klavyeleri anlık tespit eder.
   * Her klavye için bağımsız **ON / OFF** anahtarı sunar.
   * **Senaryo:** Dahili klavyenizde eksik tuşlar için Sensei devredeyken, yanına taktığınız Türkçe ISO harici klavyede çakışma olmaması için o harici klavyeyi tek tıkla pasife alabilirsiniz.

7. **⚡ Yönetilebilir Hyper Key (Caps Lock Dönüştürücü):**
   * Kullanılmayan `Caps Lock` tuşunu güçlü bir süper-değiştiriciye (Hyper Modifier) dönüştürür.
   * **Hold (Basılı Tutulduğunda):** `⌘ + ⌥ + ⌃ + ⇧` (Super Modifiers) üreterek Raycast, Alfred veya IDE kısayolları için çakışmasız süper kombinasyon sağlar.
   * **Tap (Tek Dokunulduğunda):** Elinizi uzatmadan serçe parmağınızla anında `ESC` basabilir (Vim / Terminal / Kodlama dostu) veya orijinal Caps Lock işlevini sürdürebilir.

8. **🔁 Çift Dokunma Dizilimleri (Double-Tap Sequences):**
   * Command veya Option tuşuna basmanıza gerek kalmadan, aynı tuşa iki kez hızlı basıldığında (örn. `öö` $\rightarrow$ `<`, `çç` $\rightarrow$ `>`, `..` $\rightarrow$ `|`, `--` $\rightarrow$ `~`) anında dönüştürür.
   * Yazma akışını geciktirmez (ilk harf normal basılır; ikinci basış eşiği içinde gelirse sentetik Backspace ile silinip hedef Unicode karakter enjekte edilir).

9. **🚀 macOS Başlangıç Servisi (LaunchAgent):**
   * Bilgisayarınızı yeniden başlattığınızda veya oturum açtığınızda **arka planda otomatik başlar**.
   * Web paneli veya Menü Çubuğu üzerinden tek tıkla açılıp kapatılabilir.

10. **🧪 Kapsamlı Test Kapsamı:**
    * `internal/config`, `internal/engine` ve `internal/web` modülleri için tam birim testleri içerir (`go test -v ./...`).

---

## 🔐 İlk Kurulum: macOS Erişilebilirlik İzni

macOS'in klavye tuşlarını yakalayabilmesi için uygulamanın bir kereliğe mahsus onaylanması gerekir:

1. **Sistem Ayarları** $\rightarrow$ **Gizlilik ve Güvenlik** $\rightarrow$ **Erişilebilirlik** bölümüne gidin.
2. Listede **`Keyboard Sensei`** uygulamasını bulun ve anahtarını **Açık** yapın (Eğer listede yoksa alttaki `+` butonuna basıp `/Applications/Keyboard Sensei.app` dosyasını seçin).
3. İzni verdiğiniz anda arka plan servisi otomatik olarak tuşları yakalamaya başlar (yeniden başlatma gerekmez).

---

## 🕹️ CLI Komutları (İsteğe Bağlı)

```bash
# Servisi başlangıca kur
/Applications/Keyboard\ Sensei.app/Contents/MacOS/keyboard-sensei --install

# Servisi başlangıçtan kaldır
/Applications/Keyboard\ Sensei.app/Contents/MacOS/keyboard-sensei --uninstall

# Servis durumunu kontrol et
/Applications/Keyboard\ Sensei.app/Contents/MacOS/keyboard-sensei --service-status

# Web panelini tarayıcıda aç
open http://localhost:5252
```

---

## 🖥️ Uzak Masaüstü (Microsoft Remote Desktop / RDP) ile Kullanım

Microsoft Remote Desktop (RDP) varsayılan olarak donanımsal *Scancode* modunda çalışır. Windows oturumunuzda `\` (ters slash), `<`, `>`, `|` vb. tüm özel karakterlerin sorunsuz yazılması için:

1. Uzak masaüstü penceresi açıkken üst menü çubuğundan **Connections** menüsüne gelin.
2. **Keyboard Mode** seçeneğini **"Unicode"** olarak değiştirin (veya doğrudan kısayoluna basın: **`⌃ + ⌘ + U`** / Control + Command + U).
3. Bu ayardan sonra Keyboard Sensei ile ürettiğiniz tüm karakterler Windows oturumuna kusursuz iletilecektir.

---

## 🧪 Testleri Çalıştırma

```bash
go test -v ./...
```

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"keyboard-sensei/internal/config"
	"keyboard-sensei/internal/engine"
	"keyboard-sensei/internal/service"
	"keyboard-sensei/internal/web"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	portFlag := flag.Int("port", 0, "Web arayüzü port numarası (varsayılan: 5252)")
	openBrowser := flag.Bool("open", true, "Başlangıçta tarayıcıyı otomatik aç")
	trayFlag := flag.Bool("tray", true, "macOS Menü Çubuğu (Status Bar) ikonunu göster")
	installService := flag.Bool("install", false, "macOS başlangıç servisi (LaunchAgent) olarak kur")
	uninstallService := flag.Bool("uninstall", false, "macOS başlangıç servisini kaldır")
	serviceStatus := flag.Bool("service-status", false, "Servis durumunu göster")
	flag.Parse()

	if *installService {
		fmt.Println("🚀 Keyboard Sensei macOS başlangıç servisi olarak kuruluyor...")
		if err := service.Install(""); err != nil {
			log.Fatalf("❌ Kurulum başarısız: %v\n", err)
		}
		fmt.Println("✅ Başarıyla kuruldu! Bilgisayarınız her açıldığında arka planda otomatik çalışacak.")
		fmt.Printf("📄 Servis tanımı: %s\n", service.GetPlistPath())
		fmt.Printf("📋 Log dosyası: %s\n", service.GetLogPath())
		return
	}

	if *uninstallService {
		fmt.Println("🛑 Keyboard Sensei başlangıç servisi kaldırılıyor...")
		if err := service.Uninstall(); err != nil {
			log.Fatalf("❌ Kaldırma başarısız: %v\n", err)
		}
		fmt.Println("✅ Başlangıç servisi başarıyla kaldırıldı.")
		return
	}

	if *serviceStatus {
		installed := service.IsInstalled()
		running := service.IsRunning()
		fmt.Printf("Servis Kurulu mu: %v\n", installed)
		fmt.Printf("Servis Çalışıyor mu: %v\n", running)
		fmt.Printf("Plist Yolu: %s\n", service.GetPlistPath())
		return
	}

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║               🥋 KEYBOARD SENSEI (macOS)                 ║")
	fmt.Println("║       ANSI Klavyeler İçin Türkçe Tuş Dönüştürücü         ║")
	fmt.Println("║                 Sürüm: v1.2.0                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// 1. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("⚠️  Konfigürasyon dosyası oluşturuldu/yüklendi: %v\n", err)
	}

	port := cfg.Port
	if *portFlag > 0 {
		port = *portFlag
	}

	// 2. Initialize Engine
	eng := engine.NewEngine()
	eng.UpdateRules(cfg.Rules)
	eng.SetGlobalExcludedApps(cfg.ExcludedApps)
	eng.UpdateHyperKey(cfg.HyperKey)
	eng.UpdateSequenceRules(cfg.SequenceRules, cfg.EnableSequences)
	eng.UpdateDeviceFilters(cfg.Devices)

	// 3. Check Accessibility
	if !eng.IsAccessibilityTrusted() {
		fmt.Println("⚠️  Erişilebilirlik izni henüz verilmemiş.")
		fmt.Println("👉 Sistem Ayarları > Gizlilik ve Güvenlik > Erişilebilirlik bölümünden izin verin.")
		eng.PromptAccessibility()
	} else {
		fmt.Println("✅ macOS Erişilebilirlik izni onaylı.")
	}

	// 4. Start Event Tap Engine
	if err := eng.Start(); err != nil {
		fmt.Printf("⚠️  Event Tap başlatılamadı: %v\n", err)
		fmt.Println("ℹ️  Web arayüzünden izni verdikten sonra servis otomatik aktifleşecektir.")
	} else {
		activeProfile := cfg.GetActiveProfile()
		profileName := "Varsayılan"
		if activeProfile != nil {
			profileName = activeProfile.Name
		}
		fmt.Printf("🎯 Aktif Profil: %s (%d kural aktif dinleniyor)\n", profileName, len(cfg.Rules))
	}

	// 5. Start Web Dashboard
	srv := web.NewServer(port, eng)
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("❌ Web sunucu hatası: %v", err)
		}
	}()

	url := fmt.Sprintf("http://localhost:%d", port)
	if *openBrowser {
		go func() {
			time.Sleep(400 * time.Millisecond)
			_ = exec.Command("open", url).Start()
		}()
	}

	fmt.Printf("\n🚀 Panel hazır: %s\n", url)
	if *trayFlag && cfg.ShowTrayIcon {
		fmt.Println("🍏 Menü çubuğunda (Menu Bar) 🥋 ikonu aktif.")
	}
	fmt.Println("Çıkmak için CTRL+C tuşlarına basın.")

	// 6. Graceful shutdown handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n🛑 Kapatılıyor...")
		eng.Stop()
		engine.StopMacAppLoop()
		os.Exit(0)
	}()

	if *trayFlag && cfg.ShowTrayIcon {
		runtime.LockOSThread()
		engine.RunMacAppLoop()
	} else {
		<-sigChan
		eng.Stop()
		fmt.Println("Güle güle!")
	}
}

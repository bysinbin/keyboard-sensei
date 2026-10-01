import Cocoa

func renderIcon(size: Int) -> NSImage {
    let s = CGFloat(size)
    let img = NSImage(size: NSSize(width: s, height: s))
    img.lockFocus()

    guard let ctx = NSGraphicsContext.current?.cgContext else {
        img.unlockFocus()
        return img
    }

    let margin = s * 0.08
    let corner = s * 0.22
    let rect = CGRect(x: margin, y: margin, width: s - margin * 2, height: s - margin * 2)

    // Base Squircle Background
    let path = CGPath(roundedRect: rect, cornerWidth: corner, cornerHeight: corner, transform: nil)
    ctx.addPath(path)
    ctx.clip()

    // Dark sleek gradient
    let colorSpace = CGColorSpaceCreateDeviceRGB()
    let colors = [
        NSColor(red: 0.05, green: 0.07, blue: 0.11, alpha: 1.0).cgColor,
        NSColor(red: 0.11, green: 0.15, blue: 0.22, alpha: 1.0).cgColor
    ] as CFArray
    if let gradient = CGGradient(colorsSpace: colorSpace, colors: colors, locations: [0.0, 1.0]) {
        ctx.drawLinearGradient(gradient, start: CGPoint(x: 0, y: s), end: CGPoint(x: s, y: 0), options: [])
    }

    // Border Glow
    ctx.resetClip()
    ctx.setStrokeColor(NSColor(red: 0.06, green: 0.72, blue: 0.50, alpha: 0.4).cgColor)
    ctx.setLineWidth(s * 0.02)
    ctx.addPath(path)
    ctx.strokePath()

    // Draw Keyboard Outline
    let kbRect = CGRect(x: s * 0.20, y: s * 0.26, width: s * 0.60, height: s * 0.46)
    let kbPath = CGPath(roundedRect: kbRect, cornerWidth: s * 0.06, cornerHeight: s * 0.06, transform: nil)
    ctx.setFillColor(NSColor(red: 0.15, green: 0.20, blue: 0.30, alpha: 0.7).cgColor)
    ctx.addPath(kbPath)
    ctx.fillPath()

    ctx.setStrokeColor(NSColor(red: 0.2, green: 0.8, blue: 0.6, alpha: 0.8).cgColor)
    ctx.setLineWidth(s * 0.018)
    ctx.addPath(kbPath)
    ctx.strokePath()

    // Keycaps inside keyboard
    let rows = 3
    let cols = 5
    let padX = kbRect.width * 0.06
    let padY = kbRect.height * 0.08
    let keyW = (kbRect.width - padX * CGFloat(cols + 1)) / CGFloat(cols)
    let keyH = (kbRect.height - padY * CGFloat(rows + 1)) / CGFloat(rows)

    for r in 0..<rows {
        for c in 0..<cols {
            let kx = kbRect.origin.x + padX + CGFloat(c) * (keyW + padX)
            let ky = kbRect.origin.y + padY + CGFloat(r) * (keyH + padY)
            let kRect = CGRect(x: kx, y: ky, width: keyW, height: keyH)
            let kPath = CGPath(roundedRect: kRect, cornerWidth: s * 0.015, cornerHeight: s * 0.015, transform: nil)

            if r == 1 && c == 2 {
                // Highlight middle key (Sensei key) in emerald
                ctx.setFillColor(NSColor(red: 0.06, green: 0.72, blue: 0.50, alpha: 0.9).cgColor)
            } else {
                ctx.setFillColor(NSColor(red: 0.25, green: 0.32, blue: 0.45, alpha: 0.6).cgColor)
            }
            ctx.addPath(kPath)
            ctx.fillPath()
        }
    }

    // Top Ninja Headband Accent
    let bandY = s * 0.80
    ctx.setFillColor(NSColor(red: 0.95, green: 0.25, blue: 0.25, alpha: 0.9).cgColor)
    ctx.fill(CGRect(x: margin, y: bandY, width: rect.width, height: s * 0.04))

    // Japanese Kanji / Sensei symbol text in center: "< >"
    let font = NSFont.monospacedSystemFont(ofSize: s * 0.16, weight: .black)
    let str = "< >" as NSString
    let attrs: [NSAttributedString.Key: Any] = [
        .font: font,
        .foregroundColor: NSColor.white
    ]
    let strSize = str.size(withAttributes: attrs)
    let strPoint = NSPoint(x: (s - strSize.width) / 2, y: s * 0.40)
    str.draw(at: strPoint, withAttributes: attrs)

    img.unlockFocus()
    return img
}

let fm = FileManager.default
let iconsetDir = "AppIcon.iconset"
try? fm.removeItem(atPath: iconsetDir)
try? fm.createDirectory(atPath: iconsetDir, withIntermediateDirectories: true)

let sizes: [(String, Int)] = [
    ("icon_16x16.png", 16),
    ("icon_16x16@2x.png", 32),
    ("icon_32x32.png", 32),
    ("icon_32x32@2x.png", 64),
    ("icon_128x128.png", 128),
    ("icon_128x128@2x.png", 256),
    ("icon_256x256.png", 256),
    ("icon_256x256@2x.png", 512),
    ("icon_512x512.png", 512),
    ("icon_512x512@2x.png", 1024)
]

for (filename, size) in sizes {
    let img = renderIcon(size: size)
    if let tiff = img.tiffRepresentation,
       let rep = NSBitmapImageRep(data: tiff),
       let png = rep.representation(using: .png, properties: [:]) {
        let path = "\(iconsetDir)/\(filename)"
        try? png.write(to: URL(fileURLWithPath: path))
    }
}

print("AppIcon.iconset created successfully!")

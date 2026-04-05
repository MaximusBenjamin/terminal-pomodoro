import SwiftUI

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet(charactersIn: "#"))
        var rgbValue: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&rgbValue)
        let r = Double((rgbValue & 0xFF0000) >> 16) / 255.0
        let g = Double((rgbValue & 0x00FF00) >> 8) / 255.0
        let b = Double(rgbValue & 0x0000FF) / 255.0
        self.init(red: r, green: g, blue: b)
    }
}

enum Theme {
    static let bg       = Color(hex: "#1a1b26")
    static let fg       = Color(hex: "#a9b1d6")
    static let accent   = Color(hex: "#7aa2f7")
    static let overtime = Color(hex: "#f7768e")
    static let success  = Color(hex: "#9ece6a")
    static let warning  = Color(hex: "#e0af68")
    static let border   = Color(hex: "#3b4261")
    static let muted    = Color(hex: "#565f89")

    static let heatmap0 = Color(hex: "#3b4261") // none (muted gray)
    static let heatmap1 = Color(hex: "#1e3a1e") // low
    static let heatmap2 = Color(hex: "#2d6b2d") // medium-low
    static let heatmap3 = Color(hex: "#52a852") // medium
    static let heatmap4 = Color(hex: "#9ece6a") // high (bright green)
}

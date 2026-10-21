import SwiftUI

enum AuroraTheme {
    static let background = Color.black
    static let panel = Color.white.opacity(0.08)
    static let elevated = Color.white.opacity(0.13)
    static let paper = Color(white: 0.96)
    static let mist = Color(white: 0.72)
    static let ash = Color(white: 0.52)
    static let accent = Color(red: 0.93, green: 0.29, blue: 0.18)
}

struct AuroraChipStyle: ButtonStyle {
    var selected = false
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.callout.weight(.semibold))
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .foregroundStyle(selected ? Color.black : AuroraTheme.paper)
            .background(selected ? AuroraTheme.paper : AuroraTheme.panel, in: Capsule())
            .opacity(configuration.isPressed ? 0.7 : 1)
    }
}

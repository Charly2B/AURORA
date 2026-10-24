import SwiftUI

// Direction « Cinémathèque » (maquette validée le 2026-10-24) : brun très sombre, crème, or,
// titres en serif système (New York). Les noms historiques sont gardés pour que tous les écrans suivent.
enum AuroraTheme {
    static let background = Color(red: 0.082, green: 0.067, blue: 0.055)   // #15110E
    static let sidebar = Color(red: 0.063, green: 0.051, blue: 0.043)      // #100D0B
    static let panel = cream.opacity(0.06)
    static let elevated = cream.opacity(0.10)
    static let paper = cream
    static let mist = Color(red: 0.918, green: 0.851, blue: 0.725).opacity(0.66)  // sable
    static let ash = Color(red: 0.722, green: 0.604, blue: 0.408)          // or doux #B89A68
    static let accent = Color(red: 0.851, green: 0.698, blue: 0.435)       // or #D9B26F
    static let line = accent.opacity(0.16)
    static let buttonFill = Color(red: 0.945, green: 0.894, blue: 0.800)   // #F1E4CC
    static let ink = Color(red: 0.102, green: 0.078, blue: 0.059)          // texte sur crème
    // Point « serveur connecté » ; les coches « vu » utilisent l'or.
    static let green = Color(red: 0.56, green: 0.79, blue: 0.56)

    private static let cream = Color(red: 0.957, green: 0.918, blue: 0.847) // #F4EAD8

    static func serif(_ size: CGFloat, _ weight: Font.Weight = .regular) -> Font {
        .system(size: size, weight: weight, design: .serif)
    }
}

/// Segment d'une barre de choix (dossiers, tri, saisons) : à poser dans `AuroraSegmentBar`.
struct AuroraSegmentStyle: ButtonStyle {
    var selected = false
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(AuroraTheme.serif(15))
            .lineLimit(1)
            .padding(.horizontal, 14)
            .frame(height: 30)
            .foregroundStyle(selected ? AuroraTheme.paper : AuroraTheme.mist)
            .background(selected ? AuroraTheme.accent.opacity(0.22) : .clear, in: RoundedRectangle(cornerRadius: 6))
            .contentShape(RoundedRectangle(cornerRadius: 6))
            .opacity(configuration.isPressed ? 0.7 : 1)
    }
}

/// Barre sombre au filet doré : reste lisible même posée sur un fond d'affiche clair.
struct AuroraSegmentBar<Content: View>: View {
    @ViewBuilder let content: Content
    var body: some View {
        HStack(spacing: 2) { content }
            .padding(3)
            .background(AuroraTheme.sidebar.opacity(0.88), in: RoundedRectangle(cornerRadius: 9))
            .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 9))
            .overlay(RoundedRectangle(cornerRadius: 9).strokeBorder(AuroraTheme.accent.opacity(0.22)))
            .fixedSize()
    }
}

/// Bouton principal crème (Lecture / Reprendre).
struct AuroraPrimaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 14, weight: .semibold))
            .padding(.horizontal, 20)
            .frame(height: 40)
            .foregroundStyle(AuroraTheme.ink)
            .background(AuroraTheme.buttonFill, in: RoundedRectangle(cornerRadius: 6))
            .opacity(configuration.isPressed ? 0.75 : 1)
    }
}

/// Bouton secondaire au filet doré.
struct AuroraGhostButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 13, weight: .medium))
            .padding(.horizontal, 12)
            .frame(minWidth: 40, minHeight: 40)
            .foregroundStyle(AuroraTheme.paper)
            .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(AuroraTheme.accent.opacity(0.45)))
            .contentShape(RoundedRectangle(cornerRadius: 6))
            .opacity(configuration.isPressed ? 0.6 : 1)
    }
}

struct AuroraBadge: View {
    let text: String
    var body: some View {
        Text(text.uppercased())
            .font(.system(size: 10, weight: .bold))
            .foregroundStyle(AuroraTheme.accent)
            .padding(.horizontal, 5).padding(.vertical, 1)
            .overlay(RoundedRectangle(cornerRadius: 3).strokeBorder(AuroraTheme.accent.opacity(0.5)))
    }
}

/// Petit titre en capitales espacées (« À LA UNE · FILM », « PARCOURIR »).
struct AuroraEyebrow: View {
    let text: String
    var color = AuroraTheme.accent
    var body: some View {
        Text(text.uppercased())
            .font(.system(size: 11, weight: .semibold))
            .tracking(2.4)
            .foregroundStyle(color)
    }
}

enum AuroraFormat {
    /// 166 → « 2 h 46 », 52 → « 52 min ».
    static func runtime(minutes: Int?) -> String? {
        guard let minutes, minutes > 0 else { return nil }
        return minutes < 60 ? "\(minutes) min" : "\(minutes / 60) h \(String(format: "%02d", minutes % 60))"
    }

    /// Temps restant d'une carte en cours : « 22 min restantes », « 1 h 02 restante ».
    static func remaining(_ card: Card) -> String? {
        guard card.length > 0, let position = card.position, position > 0 else { return nil }
        let minutes = Int(((card.length - position) / 60).rounded())
        guard minutes > 0 else { return nil }
        return minutes < 60 ? "\(minutes) min restantes" : "\(minutes / 60) h \(String(format: "%02d", minutes % 60)) restante"
    }

    /// « S1 · É7 » pour un épisode.
    static func episodeCode(_ card: Card) -> String? {
        guard let season = card.season, let episode = card.episode else { return nil }
        return "S\(season) · É\(episode)"
    }
}

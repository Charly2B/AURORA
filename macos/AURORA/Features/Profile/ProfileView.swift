import AppKit
import SwiftUI

struct ProfileView: View {
    @Bindable var model: ProfileModel
    let onSwitchServer: () -> Void
    let onAddServer: () -> Void
    let onChooseBackground: () -> Void
    let onLogout: () -> Void
    @AppStorage(HomeBackgroundStore.pathKey) private var backgroundPath = ""
    @AppStorage(HomeBackgroundStore.revisionKey) private var backgroundRevision = 0
    @AppStorage(HomeBackgroundStore.homeTitleSizeKey) private var homeTitleSize = 64.0
    @AppStorage(HomeBackgroundStore.homeTitleFontKey) private var homeTitleFont = "sfPro"
    @State private var backgroundError: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Image(systemName: "person.crop.circle.fill")
                .font(.system(size: 72)).foregroundStyle(AuroraTheme.mist)
                .accessibilityHidden(true)
            Text(model.username).font(.system(size: 40, weight: .bold))
            Text(model.serverAddress).foregroundStyle(AuroraTheme.mist).textSelection(.enabled)
            LabeledContent("Serveur AURORA", value: model.serverVersion)
            Text("Lecture : \(model.playbackSummary)").foregroundStyle(AuroraTheme.ash)
            if let error = model.errorMessage { Text(error).foregroundStyle(.red) }
            HStack(spacing: 12) {
                Button("Changer de NAS", action: onSwitchServer)
                Button("Ajouter un NAS", action: onAddServer)
                Button("Choisir une image de fond…", action: chooseBackground)
                Button("Déconnexion", role: .destructive, action: onLogout)
            }.buttonStyle(.bordered)
            VStack(alignment: .leading, spacing: 12) {
                Text("Titre de la page d’accueil").font(.headline)
                Text("MEDIATHEQUE AURORA")
                    .font(homeTitleFont(size: 28))
                    .tracking(2.8)
                HStack {
                    Text("Taille")
                    Slider(value: $homeTitleSize, in: 28...110, step: 1)
                        .frame(width: 220)
                    Text("\(Int(homeTitleSize)) pt").monospacedDigit().foregroundStyle(AuroraTheme.ash)
                }
                Picker("Police", selection: $homeTitleFont) {
                    Text("SF Pro").tag("sfPro")
                    Text("Helvetica").tag("helvetica")
                    Text("Avenir").tag("avenir")
                    Text("New York").tag("newYork")
                    Text("Menlo").tag("menlo")
                }
                .pickerStyle(.segmented)
                HStack {
                    if !backgroundPath.isEmpty {
                        Text("Fond personnalisé actif").font(.caption).foregroundStyle(AuroraTheme.green)
                    }
                    if let backgroundError { Text(backgroundError).font(.caption).foregroundStyle(.red) }
                }
            }
            .padding(16)
            .background(AuroraTheme.panel, in: RoundedRectangle(cornerRadius: 10))
            Spacer()
        }
        .padding(44)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .task { await model.load() }
    }

    private func homeTitleFont(size: CGFloat) -> Font {
        switch homeTitleFont {
        case "helvetica": return .custom("Helvetica Neue", size: size)
        case "avenir": return .custom("Avenir Next", size: size)
        case "newYork": return .system(size: size, weight: .regular, design: .serif)
        case "menlo": return .system(size: size, weight: .regular, design: .monospaced)
        default: return .system(size: size, weight: .light)
        }
    }

    private func chooseBackground() {
        let panel = NSOpenPanel()
        panel.allowedContentTypes = [.image]
        panel.allowsMultipleSelection = false
        panel.canChooseDirectories = false
        panel.prompt = "Choisir"
        guard panel.runModal() == .OK, let source = panel.url else { return }
        do {
            backgroundPath = try HomeBackgroundStore.copyImage(from: source).path
            backgroundRevision += 1
            backgroundError = nil
        } catch {
            backgroundError = error.localizedDescription
        }
    }
}

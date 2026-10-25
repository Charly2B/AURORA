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
    @AppStorage(HomeBackgroundStore.mastheadSizeKey) private var mastheadSize = 18.0
    @AppStorage(HomeBackgroundStore.mastheadStyleKey) private var mastheadStyle = "light"
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
                Text("Titre de la médiathèque").font(.headline)
                Text("MEDIATHEQUE AURORA")
                    .font(mastheadFont)
                    .tracking(2.8)
                HStack {
                    Text("Taille")
                    Slider(value: $mastheadSize, in: 12...30, step: 1)
                        .frame(width: 220)
                    Text("\(Int(mastheadSize)) pt").monospacedDigit().foregroundStyle(AuroraTheme.ash)
                }
                Picker("Format", selection: $mastheadStyle) {
                    Text("Sans léger").tag("light")
                    Text("Sans").tag("regular")
                    Text("Serif").tag("serif")
                    Text("Gras").tag("bold")
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

    private var mastheadFont: Font {
        switch mastheadStyle {
        case "regular": return .system(size: mastheadSize, weight: .regular)
        case "serif": return .system(size: mastheadSize, weight: .regular, design: .serif)
        case "bold": return .system(size: mastheadSize, weight: .bold)
        default: return .system(size: mastheadSize, weight: .light)
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

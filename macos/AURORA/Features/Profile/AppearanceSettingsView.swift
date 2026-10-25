import AppKit
import SwiftUI

struct AppearanceSettingsView: View {
    @AppStorage(HomeBackgroundStore.pathKey) private var backgroundPath = ""
    @State private var errorMessage: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("Apparence")
                .font(.title2.weight(.semibold))

            VStack(alignment: .leading, spacing: 12) {
                Text("Fond de la page d’accueil")
                    .font(.headline)
                backgroundPreview

                HStack {
                    Button("Choisir une image…") { chooseImage() }
                        .keyboardShortcut("o", modifiers: [.command, .option])

                    if !backgroundPath.isEmpty {
                        Button("Réinitialiser") {
                            HomeBackgroundStore.removeImage(at: backgroundPath)
                            backgroundPath = ""
                        }
                    }
                }

                Text("L’image est conservée localement sur ce Mac et utilisée comme fond de l’accueil.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .padding(16)
            .background(.quaternary.opacity(0.35), in: RoundedRectangle(cornerRadius: 12))

            if let errorMessage {
                Text(errorMessage).foregroundStyle(.red).font(.caption)
            }
        }
        .padding(20)
        .frame(width: 560, height: 430)
    }

    @ViewBuilder
    private var backgroundPreview: some View {
        if let image = NSImage(contentsOfFile: backgroundPath), !backgroundPath.isEmpty {
            Image(nsImage: image)
                .resizable()
                .aspectRatio(contentMode: .fill)
                .frame(height: 190)
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .overlay(alignment: .bottomLeading) {
                    Text("Image personnalisée")
                        .font(.caption.weight(.semibold))
                        .padding(8)
                        .background(.black.opacity(0.65), in: Capsule())
                        .padding(10)
                }
                .accessibilityLabel("Aperçu de l’image personnalisée")
        } else {
            RoundedRectangle(cornerRadius: 12)
                .fill(.quaternary)
                .frame(height: 190)
                .overlay {
                    Label("Fond AURORA par défaut", systemImage: "photo")
                        .foregroundStyle(.secondary)
                }
                .accessibilityLabel("Aperçu du fond AURORA par défaut")
        }
    }

    private func chooseImage() {
        let panel = NSOpenPanel()
        panel.allowedContentTypes = [.image]
        panel.allowsMultipleSelection = false
        panel.canChooseDirectories = false
        panel.canCreateDirectories = false
        panel.prompt = "Choisir"
        panel.message = "Choisissez l’image du fond de la page d’accueil."

        guard panel.runModal() == .OK, let source = panel.url else { return }
        do {
            backgroundPath = try HomeBackgroundStore.copyImage(from: source).path
            errorMessage = nil
        } catch {
            errorMessage = "Impossible d’enregistrer cette image : \(error.localizedDescription)"
        }
    }
}

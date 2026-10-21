import SwiftUI

// Sélecteur d'affiche / fond d'écran TMDB (administrateur). Un choix est
// téléchargé par le NAS puis appliqué : la fiche est rechargée.
struct ArtworkPickerView: View {
    @Bindable var model: DetailModel
    let onClose: () -> Void

    @State private var tab: ArtworkTab = .posters
    @State private var language: String = "all"

    enum ArtworkTab: String, CaseIterable {
        case posters = "Affiches"
        case backdrops = "Fonds"
    }

    private var choices: [ArtworkChoice] {
        guard let artwork = model.artwork else { return [] }
        let list = tab == .posters ? artwork.posters : artwork.backdrops
        guard language != "all" else { return list }
        return list.filter { language == "none" ? ($0.language ?? "").isEmpty : $0.language == language }
    }

    private var languages: [String] {
        guard let artwork = model.artwork else { return [] }
        let list = tab == .posters ? artwork.posters : artwork.backdrops
        return Array(Set(list.map { $0.language ?? "" })).sorted()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(spacing: 14) {
                Text("Affiche et fond d'écran").font(.title2.bold())
                ForEach(ArtworkTab.allCases, id: \.self) { value in
                    Button {
                        tab = value
                        language = "all"
                    } label: {
                        Text(value.rawValue).font(.callout.weight(.semibold))
                            .padding(.horizontal, 14).padding(.vertical, 7)
                            .background(tab == value ? AuroraTheme.paper : AuroraTheme.panel, in: Capsule())
                            .foregroundStyle(tab == value ? AuroraTheme.background : AuroraTheme.paper)
                    }.buttonStyle(.plain)
                }
                Spacer()
                if let artwork = model.artwork {
                    Menu {
                        Button("Toutes") { language = "all" }
                        ForEach(languages.filter { !$0.isEmpty }, id: \.self) { code in
                            Button(code.uppercased()) { language = code }
                        }
                        if languages.contains("") {
                            Button("Sans texte") { language = "none" }
                        }
                    } label: {
                        Label(language == "all" ? "Langue" : language.uppercased(), systemImage: "character.globe")
                    }
                }
                Button("Fermer") { onClose() }.buttonStyle(.bordered)
            }

            switch model.artworkState {
            case .idle, .loading:
                ProgressView("Chargement des images TMDB").frame(maxWidth: .infinity, minHeight: 260)
            case .error(let message):
                VStack(spacing: 10) {
                    Text(message).foregroundStyle(.red)
                    if message.localizedCaseInsensitiveContains("associez") {
                        Text("Ouvrez « Corriger l'identification » sur la fiche, puis revenez ici.")
                            .foregroundStyle(AuroraTheme.mist).multilineTextAlignment(.center)
                    }
                    Button("Réessayer") { Task { await model.loadArtwork() } }.buttonStyle(.bordered)
                }.frame(maxWidth: .infinity, minHeight: 200)
            case .content, .empty:
                if choices.isEmpty {
                    Text("Aucune image dans cette catégorie.").foregroundStyle(AuroraTheme.mist)
                        .frame(maxWidth: .infinity, minHeight: 200)
                } else {
                    ScrollView {
                        LazyVGrid(columns: [GridItem(.adaptive(minimum: tab == .posters ? 128 : 220), spacing: 14)], spacing: 14) {
                            ForEach(choices) { choice in
                                Button {
                                    let type = tab == .posters ? "poster" : "backdrop"
                                    Task { await model.applyArtwork(type: type, path: choice.path) }
                                } label: {
                                    VStack(spacing: 6) {
                                        AsyncImage(url: URL(string: choice.preview)) { image in
                                            image.resizable().aspectRatio(contentMode: .fill)
                                        } placeholder: {
                                            AuroraTheme.panel
                                        }
                                        .frame(maxWidth: .infinity)
                                        .frame(height: tab == .posters ? 190 : 124)
                                        .clipShape(RoundedRectangle(cornerRadius: 8))
                                    }
                                }
                                .buttonStyle(.plain)
                                .help("Appliquer cette image")
                            }
                        }
                    }
                }
            }

            if let error = model.errorMessage { Text(error).foregroundStyle(.red) }
            Spacer(minLength: 0)
        }
        .padding(24)
        .frame(width: 860, height: 620)
        .background(AuroraTheme.background)
        .foregroundStyle(AuroraTheme.paper)
        .task { await model.loadArtwork() }
    }
}
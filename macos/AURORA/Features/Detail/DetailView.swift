import SwiftUI

struct DetailView: View {
    @Bindable var model: DetailModel
    let images: ImageRepository
    let onBack: () -> Void
    let onOpen: (Card) -> Void
    let onPlay: (PlayIntent) -> Void

    @State private var showArtworkPicker = false
    @State private var showIdentificationFix = false

    var body: some View {
        ZStack(alignment: .topLeading) {
            AuroraTheme.background.ignoresSafeArea()
            if let detail = model.detail {
                // Fond du film à droite, fondu vers la gauche et vers le bas (direction Cinémathèque).
                GeometryReader { proxy in
                    AuthenticatedImage(path: detail.backdrop ?? detail.thumb, repository: images)
                        .frame(width: proxy.size.width * 0.55, height: 440)
                        .mask(LinearGradient(colors: [.clear, .black], startPoint: .leading, endPoint: UnitPoint(x: 0.45, y: 0.5)))
                        .overlay(LinearGradient(colors: [.clear, AuroraTheme.background], startPoint: UnitPoint(x: 0.5, y: 0.6), endPoint: .bottom))
                        .opacity(0.45)
                        .frame(maxWidth: .infinity, alignment: .trailing)
                }
                .ignoresSafeArea()
                .allowsHitTesting(false)

                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 30) {
                        HStack(alignment: .top, spacing: 32) {
                            ZStack(alignment: .bottomTrailing) {
                                AuthenticatedImage(path: detail.poster ?? detail.thumb, repository: images)
                                    .frame(width: 200, height: 300)
                                    .clipShape(RoundedRectangle(cornerRadius: 6))
                                    .shadow(color: .black.opacity(0.6), radius: 26, y: 18)
                                    .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(AuroraTheme.paper.opacity(0.12)))
                                if model.canEditArtwork, detail.kind == "movie" || detail.kind == "series" {
                                    Button {
                                        showArtworkPicker = true
                                    } label: {
                                        Label("Changer l'affiche", systemImage: "photo")
                                            .labelStyle(.iconOnly)
                                            .font(.body)
                                            .padding(8).background(.thinMaterial, in: Circle())
                                    }
                                    .buttonStyle(.plain)
                                    .padding(8)
                                    .help("Changer l'affiche ou le fond")
                                }
                            }
                            VStack(alignment: .leading, spacing: 11) {
                                AuroraEyebrow(text: eyebrow(detail))
                                HStack(alignment: .firstTextBaseline, spacing: 14) {
                                    Text(detail.title).font(AuroraTheme.serif(52, .semibold)).foregroundStyle(AuroraTheme.paper).lineLimit(2)
                                    if let original = detail.originalTitle, !original.isEmpty, original != detail.title {
                                        Text(original).font(AuroraTheme.serif(20)).foregroundStyle(AuroraTheme.ash).lineLimit(1)
                                    }
                                }
                                if let credit = FeatureInfo.credit(kind: detail.kind, people: detail.people) {
                                    Text(credit).font(AuroraTheme.serif(14).italic()).foregroundStyle(AuroraTheme.mist)
                                }
                                HStack(spacing: 8) {
                                    Text(metadata(detail)).foregroundStyle(AuroraTheme.mist)
                                    ForEach(badges(detail), id: \.self) { AuroraBadge(text: $0) }
                                }
                                .font(.system(size: 13))
                                if let overview = detail.overview {
                                    Text(overview).font(.system(size: 14)).lineSpacing(3).foregroundStyle(AuroraTheme.paper.opacity(0.78)).lineLimit(6)
                                }
                                MediaActionsView(model: model, onPlay: onPlay).padding(.top, 4)
                                if let seasons = detail.seasons, !seasons.isEmpty, let total = detail.total, total > 0 {
                                    let seen = max(total - (detail.unwatched ?? 0), 0)
                                    HStack(spacing: 10) {
                                        ProgressView(value: Double(seen), total: Double(total)).tint(AuroraTheme.accent).frame(width: 150)
                                        Text("\(seen) épisode\(seen > 1 ? "s" : "") vu\(seen > 1 ? "s" : "") sur \(total)").font(.system(size: 12)).foregroundStyle(AuroraTheme.ash)
                                    }
                                }
                                if detail.files.count > 1 {
                                    Menu("Version") {
                                        ForEach(detail.files) { file in
                                            Button(file.label.isEmpty ? file.name : file.label) {
                                                if let intent = model.playIntent(restart: false, fileID: file.id) { onPlay(intent) }
                                            }
                                        }
                                    }
                                    .fixedSize()
                                }
                                if model.canEditArtwork, detail.kind == "movie" || detail.kind == "series" {
                                    Button {
                                        showIdentificationFix = true
                                    } label: {
                                        Label("Corriger l'identification", systemImage: "questionmark.square.dashed")
                                    }
                                    .buttonStyle(AuroraGhostButtonStyle())
                                    .help("Chercher la bonne fiche TMDB (nécessaire avant de changer l'affiche)")
                                }
                                if let error = model.errorMessage { Text(error).foregroundStyle(.red) }
                            }.frame(maxWidth: 640, alignment: .leading)
                        }
                        SeasonEpisodesView(model: model, images: images, onPlay: onPlay)
                        PeopleRowView(people: detail.people)
                        if let cards = detail.collections, !cards.isEmpty { MediaRowView(title: "Collection", cards: cards, images: images, wide: false, onOpen: onOpen).padding(.horizontal, -36) }
                        if let cards = detail.similar, !cards.isEmpty { MediaRowView(title: "Similaires", cards: cards, images: images, wide: false, onOpen: onOpen).padding(.horizontal, -36) }
                    }.padding(.horizontal, 36).padding(.top, 58).padding(.bottom, 64)
                }
            }
            Button(action: onBack) {
                Label("Retour", systemImage: "chevron.left").font(AuroraTheme.serif(14)).foregroundStyle(AuroraTheme.accent)
            }
            .buttonStyle(.plain)
            .keyboardShortcut(.escape, modifiers: [])
            .padding(.horizontal, 36).padding(.top, 22)
        }
        .foregroundStyle(AuroraTheme.paper)
        .overlay { LoadingStateView(state: model.state, emptyMessage: "Fiche indisponible.") }
        .sheet(isPresented: $showArtworkPicker) {
            ArtworkPickerView(model: model) { showArtworkPicker = false }
        }
        .sheet(isPresented: $showIdentificationFix) {
            IdentificationFixView(model: model) { showIdentificationFix = false }
        }
        .task { await model.load() }
    }

    private func eyebrow(_ detail: Detail) -> String {
        switch detail.kind {
        case "series": ["Série", detail.year.map(String.init)].compactMap { $0 }.joined(separator: " · ")
        case "episode": [detail.seriesTitle, detail.season.map { "Saison \($0)" }, detail.episode.map { "Épisode \($0)" }].compactMap { $0 }.joined(separator: " · ")
        default: "Film"
        }
    }

    private func metadata(_ detail: Detail) -> String {
        let length = detail.kind == "series"
            ? detail.seasons.map { $0.count > 1 ? "\($0.count) saisons" : "1 saison" }
            : AuroraFormat.runtime(minutes: detail.runtime)
        return [detail.kind == "series" ? nil : detail.year.map(String.init), length, detail.genres?.first,
                detail.rating.map { String(format: "%.1f/10", $0) }]
            .compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: " · ")
    }

    private func badges(_ detail: Detail) -> [String] {
        [detail.contentRating, detail.resolution, detail.hdr].compactMap { $0 }.filter { !$0.isEmpty }
    }
}

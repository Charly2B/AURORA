import SwiftUI

struct DetailView: View {
    @Bindable var model: DetailModel
    let images: ImageRepository
    let onBack: () -> Void
    let onOpen: (Card) -> Void
    let onPlay: (PlayIntent) -> Void

    var body: some View {
        ZStack(alignment: .topLeading) {
            AuroraTheme.background.ignoresSafeArea()
            if let detail = model.detail {
                AuthenticatedImage(path: detail.backdrop ?? detail.thumb, repository: images)
                    .frame(height: 520)
                    .overlay(.linearGradient(colors: [.black.opacity(0.25), .black], startPoint: .top, endPoint: .bottom))
                    .ignoresSafeArea()
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 28) {
                        HStack(alignment: .top, spacing: 36) {
                            AuthenticatedImage(path: detail.poster ?? detail.thumb, repository: images)
                                .frame(width: 220, height: 330)
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                            VStack(alignment: .leading, spacing: 12) {
                                if detail.kind == "episode" { Text(detail.seriesTitle ?? "").foregroundStyle(AuroraTheme.accent) }
                                Text(detail.title).font(.system(size: 44, weight: .bold)).lineLimit(2)
                                Text(metadata(detail)).foregroundStyle(AuroraTheme.mist)
                                MediaActionsView(model: model, onPlay: onPlay)
                                if detail.files.count > 1 {
                                    Menu("Version") {
                                        ForEach(detail.files) { file in
                                            Button(file.label.isEmpty ? file.name : file.label) {
                                                if let intent = model.playIntent(restart: false, fileID: file.id) { onPlay(intent) }
                                            }
                                        }
                                    }
                                }
                                if let overview = detail.overview { Text(overview).font(.body).foregroundStyle(AuroraTheme.mist).lineLimit(6) }
                                if let error = model.errorMessage { Text(error).foregroundStyle(.red) }
                            }.frame(maxWidth: 850, alignment: .leading)
                        }
                        SeasonEpisodesView(model: model, images: images, onPlay: onPlay)
                        PeopleRowView(people: detail.people, images: images)
                        if let cards = detail.collections, !cards.isEmpty { MediaRowView(title: "Collection", cards: cards, images: images, wide: false, onOpen: onOpen) }
                        if let cards = detail.similar, !cards.isEmpty { MediaRowView(title: "Similaires", cards: cards, images: images, wide: false, onOpen: onOpen) }
                    }.padding(.horizontal, 54).padding(.top, 52).padding(.bottom, 64)
                }
            }
            Button(action: onBack) { Label("Retour", systemImage: "chevron.left") }
                .buttonStyle(.bordered).padding(18)
        }
        .foregroundStyle(AuroraTheme.paper)
        .overlay { LoadingStateView(state: model.state, emptyMessage: "Fiche indisponible.") }
        .task { await model.load() }
    }

    private func metadata(_ detail: Detail) -> String {
        [detail.year.map(String.init), detail.contentRating, detail.rating.map { String(format: "%.1f /10", $0) }, detail.resolution, detail.hdr]
            .compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: "   ·   ")
    }
}

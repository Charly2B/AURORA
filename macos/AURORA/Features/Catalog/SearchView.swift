import SwiftUI

struct SearchView: View {
    @Bindable var model: SearchModel
    let images: ImageRepository
    let onOpen: (Card) -> Void
    @State private var searchTask: Task<Void, Never>?

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            TextField("Un film, une série, un acteur…", text: $model.query)
                .textFieldStyle(.roundedBorder)
                .frame(maxWidth: 640)
                .padding(.horizontal, 36)
                .onChange(of: model.query) { _, query in
                    searchTask?.cancel()
                    searchTask = Task {
                        try? await Task.sleep(for: .milliseconds(300))
                        guard !Task.isCancelled else { return }
                        await model.search(query)
                    }
                }
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 24) {
                    if let label = model.fuzzyLabel { Text(label).foregroundStyle(AuroraTheme.ash).padding(.horizontal, 36) }
                    if let results = model.results {
                        if !results.movies.isEmpty { MediaRowView(title: "Films", cards: results.movies, images: images, wide: false, onOpen: onOpen) }
                        if !results.series.isEmpty { MediaRowView(title: "Séries", cards: results.series, images: images, wide: false, onOpen: onOpen) }
                        if !results.episodes.isEmpty { MediaRowView(title: "Épisodes", cards: results.episodes, images: images, wide: true, onOpen: onOpen) }
                    }
                    if model.results == nil || model.state == .empty { Text(model.prompt).foregroundStyle(AuroraTheme.mist).padding(36) }
                }
            }
            .overlay { LoadingStateView(state: model.state, emptyMessage: model.prompt) }
        }
        .padding(.top, 12)
    }
}

import SwiftUI

struct MediaActionsView: View {
    @Bindable var model: DetailModel
    let onPlay: (PlayIntent) -> Void

    var body: some View {
        if let detail = model.detail {
            HStack(spacing: 10) {
                if let intent = model.playIntent(restart: false, fileID: nil) {
                    let resuming = detail.state.position > 0 && !detail.state.watched
                    Button { onPlay(intent) } label: { Label(resuming ? "Reprendre" : "Lecture", systemImage: "play.fill") }
                        .buttonStyle(AuroraPrimaryButtonStyle())
                    if resuming, let restart = model.playIntent(restart: true, fileID: intent.fileID) {
                        Button { onPlay(restart) } label: { Label("Depuis le début", systemImage: "backward.end") }
                            .buttonStyle(AuroraGhostButtonStyle())
                    }
                }
                // Série : « vu » se règle par épisode (menu « ⋯ »), pas de bouton global (demande du 2026-10-25).
                if detail.kind != "series" {
                    Button { Task { await model.setWatched(!detail.state.watched) } } label: {
                        Label(detail.state.watched ? "Non vu" : "Vu", systemImage: detail.state.watched ? "eye.slash" : "checkmark")
                    }
                }
                Button { Task { await model.toggleWatchlist() } } label: {
                    Image(systemName: detail.state.watchlist ? "text.badge.checkmark" : "text.badge.plus")
                }
                .help(detail.state.watchlist ? "Retirer de Ma liste" : "Ajouter à Ma liste")
                .accessibilityLabel(detail.state.watchlist ? "Retirer de Ma liste" : "Ajouter à Ma liste")
                Button { Task { await model.toggleFavorite() } } label: {
                    Image(systemName: detail.state.favorite ? "heart.fill" : "heart")
                        .foregroundStyle(detail.state.favorite ? AuroraTheme.accent : AuroraTheme.paper)
                }
                .help(detail.state.favorite ? "Retirer des favoris" : "Ajouter aux favoris")
                .accessibilityLabel(detail.state.favorite ? "Retirer des favoris" : "Ajouter aux favoris")
            }
            .buttonStyle(AuroraGhostButtonStyle())
        }
    }
}

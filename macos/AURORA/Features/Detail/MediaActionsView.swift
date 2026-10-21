import SwiftUI

struct MediaActionsView: View {
    @Bindable var model: DetailModel
    let onPlay: (PlayIntent) -> Void

    var body: some View {
        if let detail = model.detail {
            HStack(spacing: 12) {
                if let intent = model.playIntent(restart: false, fileID: nil) {
                    Button(detail.state.position > 0 && !detail.state.watched ? "▶  Reprendre" : "▶  Lecture") { onPlay(intent) }
                        .buttonStyle(.borderedProminent)
                    if detail.state.position > 0 && !detail.state.watched,
                       let restart = model.playIntent(restart: true, fileID: intent.fileID) {
                        Button("Depuis le début") { onPlay(restart) }
                    }
                }
                Button(detail.state.watched ? "Non vu" : "Vu") { Task { await model.setWatched(!detail.state.watched) } }
                Button(detail.state.watchlist ? "✓ Ma liste" : "+ Ma liste") { Task { await model.toggleWatchlist() } }
                Button(detail.state.favorite ? "♥ Favori" : "♡ Favori") { Task { await model.toggleFavorite() } }
            }
            .buttonStyle(.bordered)
        }
    }
}

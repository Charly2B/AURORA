import SwiftUI

struct NextEpisodeCountdown {
    private(set) var remaining: Int?

    init(remaining: Int?) { self.remaining = remaining }

    mutating func tick() -> Bool {
        guard let remaining else { return false }
        if remaining <= 1 {
            self.remaining = 0
            return true
        }
        self.remaining = remaining - 1
        return false
    }

    mutating func cancel() { remaining = nil }
}

struct NextEpisodeView: View {
    let card: Card
    let countdown: Int?
    let images: ImageRepository
    let onPlay: () -> Void
    let onCancel: () -> Void

    var body: some View {
        HStack(spacing: 16) {
            AuthenticatedImage(path: card.thumb ?? card.backdrop, repository: images)
                .frame(width: 180, height: 100)
                .clipShape(RoundedRectangle(cornerRadius: 8))
            VStack(alignment: .leading, spacing: 8) {
                Text("Épisode suivant").font(.caption.bold()).foregroundStyle(AuroraTheme.ash)
                Text(card.title).font(.headline).lineLimit(2)
                HStack {
                    Button(countdown.map { "Lecture dans \($0)" } ?? "Lire") { onPlay() }
                        .buttonStyle(.borderedProminent)
                    Button("Annuler", action: onCancel)
                }
            }
        }
        .padding(16)
        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 14))
        .frame(width: 470)
    }
}

import SwiftUI

struct MediaCardView: View {
    let card: Card
    let images: ImageRepository
    let onOpen: () -> Void
    var wide = false
    // Optionnel : prévient la page quand la carte est survolée (fond ambiance, titre à la une).
    var onAmbientHover: ((Card?) -> Void)? = nil
    var highlighted = false

    @State private var hovered = false

    var body: some View {
        Button(action: onOpen) {
            VStack(alignment: .leading, spacing: 7) {
                ZStack(alignment: .bottomLeading) {
                    AuthenticatedImage(path: wide ? card.backdrop ?? card.thumb : card.poster ?? card.thumb, repository: images)
                        .aspectRatio(wide ? 16 / 9 : 2 / 3, contentMode: .fit)
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                    if card.progress > 0 {
                        GeometryReader { proxy in
                            VStack { Spacer()
                                ZStack(alignment: .leading) {
                                    Capsule().fill(.white.opacity(0.25))
                                    Capsule().fill(AuroraTheme.accent).frame(width: (proxy.size.width - 18) * card.progress)
                                }
                                .frame(height: 3)
                                .padding(.horizontal, 9).padding(.bottom, 8)
                            }
                        }
                    }
                }
                .overlay(alignment: .topTrailing) {
                    if card.watched {
                        Image(systemName: "checkmark")
                            .font(.system(size: 11, weight: .bold))
                            .foregroundStyle(.white)
                            .frame(width: 22, height: 22)
                            .background(AuroraTheme.seen, in: Circle())
                            .padding(7)
                    } else if let count = card.unwatched, count > 0 {
                        Text("\(count)").font(.caption.bold()).foregroundStyle(AuroraTheme.ink)
                            .padding(.horizontal, 7).frame(minWidth: 22, minHeight: 22)
                            .background(AuroraTheme.accent, in: Capsule()).padding(7)
                    }
                }
                .overlay {
                    if highlighted {
                        RoundedRectangle(cornerRadius: 5).strokeBorder(AuroraTheme.accent, lineWidth: 2).padding(-3)
                    }
                }
                Text(wide && card.kind == "episode" ? card.seriesTitle ?? card.title : card.title)
                    .font(AuroraTheme.serif(14))
                    .foregroundStyle(AuroraTheme.paper)
                    .lineLimit(1)
                if let subtitle {
                    Text(subtitle).font(.caption).foregroundStyle(AuroraTheme.ash).lineLimit(1)
                }
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .offset(y: hovered ? -3 : 0)
        .animation(.easeOut(duration: 0.15), value: hovered)
        .onHover { entered in
            hovered = entered
            if let onAmbientHover { onAmbientHover(entered ? card : nil) }
        }
        .accessibilityLabel(card.title)
    }

    private var subtitle: String? {
        if wide, card.kind == "episode" {
            return [AuroraFormat.episodeCode(card), AuroraFormat.remaining(card)].compactMap { $0 }.joined(separator: " · ")
        }
        if wide, let remaining = AuroraFormat.remaining(card) { return remaining }
        return card.year.map(String.init)
    }
}

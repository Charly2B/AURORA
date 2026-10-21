import SwiftUI

struct MediaCardView: View {
    let card: Card
    let images: ImageRepository
    let onOpen: () -> Void
    var wide = false

    @State private var hovered = false

    var body: some View {
        Button(action: onOpen) {
            VStack(alignment: .leading, spacing: 8) {
                ZStack(alignment: .bottomLeading) {
                    AuthenticatedImage(path: wide ? card.backdrop ?? card.thumb : card.poster ?? card.thumb, repository: images)
                        .aspectRatio(wide ? 16 / 9 : 2 / 3, contentMode: .fit)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                    if card.progress > 0 {
                        GeometryReader { proxy in
                            VStack { Spacer()
                                Rectangle().fill(AuroraTheme.accent)
                                    .frame(width: proxy.size.width * card.progress, height: 4)
                            }
                        }
                    }
                    if card.watched {
                        Image(systemName: "checkmark.circle.fill")
                            .padding(8).foregroundStyle(.white)
                    } else if let count = card.unwatched, count > 0 {
                        Text("\(count)").font(.caption.bold()).padding(6)
                            .background(AuroraTheme.accent, in: Circle()).padding(8)
                    }
                }
                Text(card.title)
                    .font(.callout.weight(.semibold))
                    .lineLimit(2)
                if let year = card.year { Text(String(year)).font(.caption).foregroundStyle(AuroraTheme.ash) }
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .scaleEffect(hovered ? 1.035 : 1)
        .animation(.easeOut(duration: 0.15), value: hovered)
        .onHover { hovered = $0 }
        .accessibilityLabel(card.title)
    }
}

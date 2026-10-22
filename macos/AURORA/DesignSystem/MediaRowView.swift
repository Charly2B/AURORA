import SwiftUI

struct MediaRowView: View {
    let title: String
    let cards: [Card]
    let images: ImageRepository
    let wide: Bool
    let onOpen: (Card) -> Void
    var onHideContinue: ((Card) -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title).font(.title2.bold()).padding(.horizontal, 36)
            ScrollView(.horizontal) {
                LazyHStack(alignment: .top, spacing: 18) {
                    ForEach(cards) { card in
                        MediaCardView(card: card, images: images, onOpen: { onOpen(card) }, wide: wide)
                            .frame(width: wide ? 300 : 170)
                            .overlay(alignment: .topTrailing) {
                                // Bouton visible pour retirer l'œuvre de « Continuer à regarder »
                                // (le clic droit reste disponible en secours).
                                if let onHideContinue {
                                    Button {
                                        onHideContinue(card)
                                    } label: {
                                        Image(systemName: "xmark.circle.fill")
                                            .font(.system(size: 20))
                                            .foregroundStyle(.white, .black.opacity(0.6))
                                            .padding(6)
                                            .contentShape(Circle())
                                    }
                                    .buttonStyle(.plain)
                                    .help("Retirer de Continuer à regarder")
                                    .accessibilityLabel("Retirer \(card.title) de Continuer à regarder")
                                }
                            }
                            .contextMenu {
                                if let onHideContinue {
                                    Button("Retirer de Continuer à regarder", role: .destructive) { onHideContinue(card) }
                                }
                            }
                    }
                }
                .padding(.horizontal, 36).padding(.bottom, 8)
            }
            .scrollIndicators(.hidden)
        }
    }
}

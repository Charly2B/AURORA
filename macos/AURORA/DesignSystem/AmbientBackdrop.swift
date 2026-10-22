import SwiftUI

// Fond « ambiance » des pages Films/Séries : backdrop d'une œuvre de la liste,
// légèrement flouté et fondu en noir vers le bas pour rester lisible sous la grille.
struct AmbientBackdrop: View {
    let card: Card?
    let images: ImageRepository

    var body: some View {
        GeometryReader { proxy in
            AuthenticatedImage(path: card?.backdrop ?? card?.thumb, repository: images)
                .frame(width: proxy.size.width, height: proxy.size.height)
                .scaleEffect(1.12) // cache les bords délavés du flou
                .blur(radius: 14)
                .opacity(0.8)
                .overlay(
                    LinearGradient(
                        stops: [
                            .init(color: .clear, location: 0),
                            .init(color: AuroraTheme.background.opacity(0.35), location: 0.55),
                            .init(color: AuroraTheme.background, location: 0.97)
                        ],
                        startPoint: .top, endPoint: .bottom
                    )
                )
                .clipped()
                .allowsHitTesting(false)
        }
        .frame(height: 440)
    }
}
import SwiftUI

// Fond « ambiance » des pages Films/Séries : grande image (featured ou carte survolée),
// floutée et fondu en noir pour rester discrète derrière la grille. Façon Plex / Apple TV.
struct AmbientBackdrop: View {
    let card: Card?
    let images: ImageRepository

    var body: some View {
        GeometryReader { proxy in
            AuthenticatedImage(path: card?.backdrop ?? card?.thumb, repository: images)
                .frame(width: proxy.size.width, height: proxy.size.height)
                .opacity(0.45)
                .blur(radius: 20)
                .scaleEffect(1.15) // cache les bords délavés produits par le flou
                .overlay(
                    LinearGradient(
                        colors: [AuroraTheme.background.opacity(0.35), AuroraTheme.background],
                        startPoint: .top, endPoint: .bottom
                    )
                )
                .clipped()
                .allowsHitTesting(false)
        }
        .frame(height: 400)
    }
}
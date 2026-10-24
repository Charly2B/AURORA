import SwiftUI

// Fond « ambiance » des pages Films/Séries : backdrop de l'œuvre survolée (sinon la première),
// sur toute la page, flouté et assombri pour que la grille reste lisible.
struct AmbientBackdrop: View {
    let card: Card?
    let images: ImageRepository

    var body: some View {
        GeometryReader { proxy in
            AuthenticatedImage(path: card?.backdrop ?? card?.thumb, repository: images)
                .frame(width: proxy.size.width, height: proxy.size.height)
                .scaleEffect(1.12) // cache les bords délavés du flou
                .blur(radius: 24)
                .overlay(
                    LinearGradient(
                        stops: [
                            .init(color: AuroraTheme.background.opacity(0.45), location: 0),
                            .init(color: AuroraTheme.background.opacity(0.62), location: 0.5),
                            .init(color: AuroraTheme.background.opacity(0.82), location: 1)
                        ],
                        startPoint: .top, endPoint: .bottom
                    )
                )
                .clipped()
                .allowsHitTesting(false)
        }
        .ignoresSafeArea()
    }
}

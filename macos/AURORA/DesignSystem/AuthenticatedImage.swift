import AppKit
import SwiftUI

struct AuthenticatedImage: View {
    let path: String?
    let repository: ImageRepository
    var contentMode: ContentMode = .fill

    @State private var image: NSImage?

    var body: some View {
        Group {
            if let image {
                Image(nsImage: image)
                    .resizable()
                    .aspectRatio(contentMode: contentMode)
            } else {
                ZStack {
                    AuroraTheme.elevated
                    Image(systemName: "film")
                        .font(.largeTitle)
                        .foregroundStyle(AuroraTheme.ash)
                }
            }
        }
        .clipped()
        .task(id: path) {
            guard let path else { image = nil; return }
            guard let data = try? await repository.data(for: path) else { return }
            image = NSImage(data: data)
        }
    }
}

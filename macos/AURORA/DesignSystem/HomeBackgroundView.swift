import AppKit
import SwiftUI

struct HomeBackgroundView: View {
    let path: String

    var body: some View {
        GeometryReader { geometry in
            if let image = NSImage(contentsOfFile: path), !path.isEmpty {
                Image(nsImage: image)
                    .resizable()
                    .scaledToFill()
                    .frame(width: geometry.size.width, height: geometry.size.height)
                    .clipped()
                    .overlay(.black.opacity(0.2))
                    .overlay {
                        LinearGradient(
                            colors: [.black.opacity(0.18), .black.opacity(0.78)],
                            startPoint: .top,
                            endPoint: .bottom
                        )
                    }
                    .accessibilityHidden(true)
            } else {
                Color.clear
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .ignoresSafeArea()
    }
}

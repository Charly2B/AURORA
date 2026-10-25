import AppKit
import SwiftUI

struct HomeBackgroundView: View {
    let path: String

    var body: some View {
        if let image = NSImage(contentsOfFile: path), !path.isEmpty {
            Image(nsImage: image)
                .resizable()
                .aspectRatio(contentMode: .fill)
                .overlay(.black.opacity(0.2))
                .overlay {
                    LinearGradient(
                        colors: [.black.opacity(0.18), .black.opacity(0.78)],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                }
                .ignoresSafeArea()
                .accessibilityHidden(true)
        } else {
            Color.clear
        }
    }
}

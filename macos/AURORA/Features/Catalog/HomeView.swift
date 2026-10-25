import SwiftUI

enum HomePresentation {
    static let title = "MEDIATHEQUE AURORA"
    static let showsMediaThumbnails = false
}

struct HomeView: View {
    @Bindable var model: HomeModel
    @AppStorage(HomeBackgroundStore.pathKey) private var customBackgroundPath = ""

    var body: some View {
        ZStack {
            if !customBackgroundPath.isEmpty {
                HomeBackgroundView(path: customBackgroundPath)
            }

            GeometryReader { geometry in
                Text(HomePresentation.title)
                    .font(.system(size: titleSize(for: geometry.size.width), weight: .light))
                    .tracking(6)
                    .foregroundStyle(.white)
                    .lineLimit(1)
                    .minimumScaleFactor(0.5)
                    .shadow(color: .black.opacity(0.65), radius: 18, y: 4)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .padding(.horizontal, 36)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .task { await model.load() }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(HomePresentation.title)
    }

    private func titleSize(for width: CGFloat) -> CGFloat {
        min(max(width * 0.055, 28), 64)
    }
}

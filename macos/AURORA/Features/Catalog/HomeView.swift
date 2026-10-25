import SwiftUI

enum HomePresentation {
    static let title = "MEDIATHEQUE AURORA"
    static let showsMediaThumbnails = false
}

struct HomeView: View {
    @Bindable var model: HomeModel
    @AppStorage(HomeBackgroundStore.homeTitleSizeKey) private var titleSizeSetting = 64.0
    @AppStorage(HomeBackgroundStore.homeTitleFontKey) private var titleFont = "sfPro"

    var body: some View {
        GeometryReader { geometry in
            Text(HomePresentation.title)
                .font(homeTitleFont(size: titleSize(for: geometry.size.width)))
                .tracking(6)
                .foregroundStyle(.white)
                .lineLimit(1)
                .minimumScaleFactor(0.5)
                .shadow(color: .black.opacity(0.65), radius: 18, y: 4)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .padding(.horizontal, 36)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .task { await model.load() }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(HomePresentation.title)
    }

    private func titleSize(for width: CGFloat) -> CGFloat {
        min(CGFloat(titleSizeSetting), max(width - 72, 28))
    }

    private func homeTitleFont(size: CGFloat) -> Font {
        switch titleFont {
        case "helvetica": return .custom("Helvetica Neue", size: size)
        case "avenir": return .custom("Avenir Next", size: size)
        case "newYork": return .system(size: size, weight: .regular, design: .serif)
        case "menlo": return .system(size: size, weight: .regular, design: .monospaced)
        default: return .system(size: size, weight: .light)
        }
    }
}

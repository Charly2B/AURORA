import SwiftUI

struct LoadingStateView: View {
    let state: CatalogLoadState
    let emptyMessage: String

    var body: some View {
        switch state {
        case .loading:
            ProgressView("Chargement…").frame(maxWidth: .infinity, maxHeight: .infinity)
        case .empty:
            Text(emptyMessage).foregroundStyle(AuroraTheme.mist).frame(maxWidth: .infinity, maxHeight: .infinity)
        case .error(let message):
            Text(message).foregroundStyle(.red).frame(maxWidth: .infinity, maxHeight: .infinity)
        default:
            EmptyView()
        }
    }
}

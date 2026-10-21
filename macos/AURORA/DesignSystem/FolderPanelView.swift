import SwiftUI

struct FolderPanelView: View {
    let folder: FolderContent
    let kind: MediaKind
    let images: ImageRepository
    let onOpen: (Card) -> Void

    private let columns = [GridItem(.adaptive(minimum: 150, maximum: 220), spacing: 20)]

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 8) {
                Circle().fill(Color(red: 1, green: 0.37, blue: 0.34)).frame(width: 12, height: 12)
                Circle().fill(Color(red: 1, green: 0.74, blue: 0.18)).frame(width: 12, height: 12)
                Circle().fill(Color(red: 0.16, green: 0.78, blue: 0.25)).frame(width: 12, height: 12)
                Text(folder.library.name).font(.title2.bold()).padding(.leading, 8)
                Text("\(folder.items.count) \(kind == .movie ? "film" : "série")\(folder.items.count > 1 ? "s" : "")")
                    .foregroundStyle(AuroraTheme.ash)
                Spacer()
            }
            .padding(16)
            .background(AuroraTheme.elevated)
            LazyVGrid(columns: columns, alignment: .leading, spacing: 22) {
                ForEach(folder.items) { card in
                    MediaCardView(card: card, images: images, onOpen: { onOpen(card) })
                }
            }
            .padding(20)
            .background(AuroraTheme.panel)
        }
        .clipShape(RoundedRectangle(cornerRadius: 16))
    }
}

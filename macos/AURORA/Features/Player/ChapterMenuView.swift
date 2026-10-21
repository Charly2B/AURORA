import SwiftUI

struct ChapterMenuView: View {
    let chapters: [Chapter]
    let onSelect: (Double) -> Void
    let onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack { Text("Chapitres").font(.headline); Spacer(); Button(action: onClose) { Image(systemName: "xmark") }.buttonStyle(.plain) }
            Divider()
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 10) {
                    ForEach(chapters) { chapter in
                        Button {
                            onSelect(chapter.start)
                            onClose()
                        } label: {
                            HStack { Text(chapter.title.isEmpty ? "Chapitre" : chapter.title); Spacer(); Text(Self.clock(chapter.start)).monospacedDigit() }
                        }.buttonStyle(.plain)
                    }
                }
            }.frame(maxHeight: 300)
        }
        .padding(16)
        .frame(width: 350)
        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 14))
    }

    static func clock(_ seconds: Double) -> String {
        let value = max(Int(seconds), 0)
        return String(format: "%d:%02d:%02d", value / 3600, (value % 3600) / 60, value % 60)
    }
}

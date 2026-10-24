import SwiftUI

struct ChapterMenuView: View {
    let chapters: [Chapter]
    let onSelect: (Double) -> Void
    let onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack { Text("Chapitres").font(AuroraTheme.serif(16, .semibold)).foregroundStyle(AuroraTheme.paper); Spacer(); Button(action: onClose) { Image(systemName: "xmark") }.buttonStyle(.plain) }
            Rectangle().fill(AuroraTheme.line).frame(height: 1)
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 10) {
                    ForEach(chapters) { chapter in
                        Button {
                            onSelect(chapter.start)
                            onClose()
                        } label: {
                            HStack { Text(chapter.title.isEmpty ? "Chapitre" : chapter.title); Spacer(); Text(Self.clock(chapter.start)).monospacedDigit().foregroundStyle(AuroraTheme.ash) }.font(.system(size: 13)).foregroundStyle(AuroraTheme.paper)
                        }.buttonStyle(.plain)
                    }
                }
            }.frame(maxHeight: 300)
        }
        .padding(16)
        .frame(width: 350)
        .background(AuroraTheme.background.opacity(0.95), in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(AuroraTheme.line))
        .shadow(color: .black.opacity(0.5), radius: 24, y: 12)
    }

    static func clock(_ seconds: Double) -> String {
        let value = max(Int(seconds), 0)
        return String(format: "%d:%02d:%02d", value / 3600, (value % 3600) / 60, value % 60)
    }
}

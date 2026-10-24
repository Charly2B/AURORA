import SwiftUI

// Distribution en texte, façon générique : « Hiroyuki Sanada — Toranaga ».
struct PeopleRowView: View {
    let people: [Person]

    var body: some View {
        let cast = people.filter { $0.role == "actor" }.prefix(10)
        if !cast.isEmpty {
            VStack(alignment: .leading, spacing: 12) {
                Rectangle().fill(AuroraTheme.line).frame(height: 1)
                Text("Distribution").font(AuroraTheme.serif(19, .semibold)).foregroundStyle(AuroraTheme.paper)
                ScrollView(.horizontal) {
                    HStack(alignment: .firstTextBaseline, spacing: 30) {
                        ForEach(Array(cast)) { person in
                            HStack(spacing: 4) {
                                Text(person.name).foregroundStyle(AuroraTheme.paper.opacity(0.8))
                                if let character = person.character, !character.isEmpty {
                                    Text("— \(character)").italic().foregroundStyle(AuroraTheme.ash)
                                }
                            }
                            .font(AuroraTheme.serif(14))
                            .lineLimit(1)
                        }
                    }
                }
                .scrollIndicators(.hidden)
            }
        }
    }
}

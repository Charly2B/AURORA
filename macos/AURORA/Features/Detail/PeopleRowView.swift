import SwiftUI

struct PeopleRowView: View {
    let people: [Person]
    let images: ImageRepository

    var body: some View {
        if !people.isEmpty {
            VStack(alignment: .leading, spacing: 12) {
                Text("Distribution").font(.title2.bold())
                ScrollView(.horizontal) {
                    LazyHStack(spacing: 18) {
                        ForEach(people) { person in
                            VStack(alignment: .leading, spacing: 8) {
                                AuthenticatedImage(path: person.photo, repository: images)
                                    .frame(width: 110, height: 140)
                                    .clipShape(RoundedRectangle(cornerRadius: 10))
                                Text(person.name).font(.callout.weight(.semibold)).lineLimit(1)
                                Text(person.character ?? person.role).font(.caption).foregroundStyle(AuroraTheme.ash).lineLimit(1)
                            }.frame(width: 110, alignment: .leading)
                        }
                    }
                }.scrollIndicators(.hidden)
            }
        }
    }
}

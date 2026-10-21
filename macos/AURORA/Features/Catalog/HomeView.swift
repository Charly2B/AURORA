import SwiftUI

struct HomeView: View {
    @Bindable var model: HomeModel
    let images: ImageRepository
    let onOpen: (Card) -> Void
    let onPlay: (Card) -> Void
    var onHideContinue: (Card) -> Void = { _ in }
    @State private var pendingRemoval: Card?

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 28) {
                if let hero = model.hero {
                    ZStack(alignment: .bottomLeading) {
                        AuthenticatedImage(path: hero.backdrop ?? hero.thumb, repository: images)
                            .frame(height: 390)
                            .overlay(.linearGradient(colors: [.clear, .black], startPoint: .center, endPoint: .bottom))
                        VStack(alignment: .leading, spacing: 12) {
                            Text(hero.title).font(.system(size: 40, weight: .bold))
                            Button(hero.progress > 0 ? "▶  Reprendre" : "▶  Lecture") { onPlay(hero) }
                                .buttonStyle(.borderedProminent)
                        }.padding(36)
                    }
                }
                ForEach(model.sections) { section in
                    MediaRowView(
                        title: section.title,
                        cards: section.items,
                        images: images,
                        wide: section.id == "continue" || section.id == "nextup",
                        onOpen: onOpen,
                        onHideContinue: section.id == "continue" ? { pendingRemoval = $0 } : nil
                    )
                }
            }
        }
        .overlay { LoadingStateView(state: model.state, emptyMessage: "Aucune vidéo pour l’instant.") }
        .task { await model.load() }
        .confirmationDialog("Retirer de Continuer à regarder ?", isPresented: Binding(
            get: { pendingRemoval != nil },
            set: { if !$0 { pendingRemoval = nil } }
        )) {
            Button("Retirer", role: .destructive) {
                if let card = pendingRemoval { onHideContinue(card) }
                pendingRemoval = nil
            }
            Button("Annuler", role: .cancel) { pendingRemoval = nil }
        } message: {
            Text("Le fichier et la progression sont conservés.")
        }
    }
}

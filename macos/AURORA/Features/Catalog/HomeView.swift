import SwiftUI

struct HomeView: View {
    @Bindable var model: HomeModel
    let images: ImageRepository
    let onOpen: (Card) -> Void
    let onPlay: (Card) -> Void
    var onHideContinue: (Card) -> Void = { _ in }
    @State private var pendingRemoval: Card?
    @State private var hovered: Card?

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 30) {
                if let card = model.featured, let info = model.info[card.id] {
                    FeaturedHeader(card: card, info: info, images: images, onPlay: { onPlay(card) }, onOpen: { onOpen(card) })
                        .id(card.id)
                        .transition(.opacity)
                }
                ForEach(model.sections) { section in
                    MediaRowView(
                        title: section.title,
                        cards: section.items,
                        images: images,
                        wide: section.id == "continue" || section.id == "nextup",
                        onOpen: onOpen,
                        onHideContinue: section.id == "continue" ? { pendingRemoval = $0 } : nil,
                        highlightedID: model.featured?.id,
                        onHover: { hovered = $0 }
                    )
                }
            }
            .padding(.top, 28).padding(.bottom, 36)
            .animation(.easeOut(duration: 0.2), value: model.featured?.id)
        }
        .background(AuroraTheme.background)
        .overlay { LoadingStateView(state: model.state, emptyMessage: "Aucune vidéo pour l’instant.") }
        .task { await model.load() }
        // Petit délai : traverser une rangée à la souris ne fait pas défiler tous les titres.
        .task(id: hovered?.id) {
            guard let card = hovered else { return }
            try? await Task.sleep(for: .milliseconds(160))
            guard !Task.isCancelled else { return }
            model.feature(card)
        }
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

/// Carte « À la une » : affiche, titre en serif, crédits, résumé, Lecture ; le fond du film en fondu à droite.
private struct FeaturedHeader: View {
    let card: Card
    let info: FeatureInfo
    let images: ImageRepository
    let onPlay: () -> Void
    let onOpen: () -> Void

    var body: some View {
        HStack(alignment: .top, spacing: 32) {
            AuthenticatedImage(path: info.poster ?? card.thumb, repository: images)
                .frame(width: 196, height: 294)
                .clipShape(RoundedRectangle(cornerRadius: 6))
                .shadow(color: .black.opacity(0.6), radius: 26, y: 18)
                .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(AuroraTheme.paper.opacity(0.12)))
                .onTapGesture(perform: onOpen)

            VStack(alignment: .leading, spacing: 12) {
                AuroraEyebrow(text: "À la une · \(info.eyebrow)")
                Text(info.title)
                    .font(AuroraTheme.serif(46, .semibold))
                    .foregroundStyle(AuroraTheme.paper)
                    .lineLimit(2)
                if let credit = info.credit {
                    Text(credit).font(AuroraTheme.serif(14).italic()).foregroundStyle(AuroraTheme.mist).lineLimit(2)
                }
                HStack(spacing: 8) {
                    Text(metadata).foregroundStyle(AuroraTheme.mist)
                    ForEach(badges, id: \.self) { AuroraBadge(text: $0) }
                }
                .font(.system(size: 13))
                if let overview = info.overview, !overview.isEmpty {
                    Text(overview)
                        .font(.system(size: 14))
                        .lineSpacing(3)
                        .foregroundStyle(AuroraTheme.paper.opacity(0.75))
                        .lineLimit(3)
                        .frame(maxWidth: 520, alignment: .leading)
                }
                HStack(spacing: 10) {
                    Button(action: onPlay) {
                        Label((card.position ?? 0) > 0 && !card.watched ? "Reprendre" : "Lecture", systemImage: "play.fill")
                    }
                    .buttonStyle(AuroraPrimaryButtonStyle())
                    Button(action: onOpen) { Label("Fiche", systemImage: "info.circle") }
                        .buttonStyle(AuroraGhostButtonStyle())
                }
                .padding(.top, 4)
            }
            .padding(.top, 8)
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 36)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(alignment: .trailing) {
            // Fond du film, masqué vers la gauche pour laisser le texte lisible.
            AuthenticatedImage(path: info.backdrop, repository: images)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .mask(LinearGradient(colors: [.clear, .black], startPoint: .leading, endPoint: UnitPoint(x: 0.45, y: 0.5)))
                .opacity(0.5)
                .containerRelativeFrame(.horizontal) { width, _ in width * 0.62 }
                .clipShape(RoundedRectangle(cornerRadius: 4))
                .padding(.trailing, 36)
                .allowsHitTesting(false)
        }
    }

    private var metadata: String {
        [card.year.map(String.init), AuroraFormat.runtime(minutes: card.runtime), info.genres.first]
            .compactMap { $0 }.joined(separator: " · ")
    }

    private var badges: [String] {
        [card.resolution, card.hdr].compactMap { $0 }.filter { !$0.isEmpty }
    }
}

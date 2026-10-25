import SwiftUI

struct SeasonEpisodesView: View {
    @Bindable var model: DetailModel
    let images: ImageRepository
    let onPlay: (PlayIntent) -> Void

    var body: some View {
        if let seasons = model.detail?.seasons, !seasons.isEmpty {
            VStack(alignment: .leading, spacing: 14) {
                HStack(alignment: .firstTextBaseline, spacing: 10) {
                    ScrollView(.horizontal) {
                        AuroraSegmentBar {
                            ForEach(seasons) { season in
                                Button(season.title) { Task { await model.selectSeason(season.number) } }
                                    .buttonStyle(AuroraSegmentStyle(selected: model.selectedSeason == season.number))
                            }
                        }
                    }
                    .scrollIndicators(.hidden)
                    .fixedSize(horizontal: false, vertical: true)
                    if let count = seasons.first(where: { $0.number == model.selectedSeason })?.episodes, count > 0 {
                        Text("\(count) épisodes").font(.system(size: 13)).foregroundStyle(AuroraTheme.ash).fixedSize()
                    }
                }

                // Liste verticale (la fiche défile déjà) : vignette à gauche, résumé à droite.
                LazyVStack(alignment: .leading, spacing: 6) {
                    ForEach(model.episodes) { episode in
                        EpisodeCardView(model: model, episode: episode, current: episode.id == currentID, images: images, onPlay: onPlay)
                    }
                }
                .padding(.horizontal, -8)
                .overlay { LoadingStateView(state: model.episodesState, emptyMessage: "Aucun épisode.") }
            }
        }
    }

    /// Épisode à reprendre (en cours), sinon le premier non vu : entouré d'or.
    private var currentID: Int64? {
        (model.episodes.first { ($0.position ?? 0) > 0 && !$0.watched } ?? model.episodes.first { !$0.watched })?.id
    }
}

// Une ligne d'épisode, à la manière de Plex : un clic lance la lecture ; « vu » se règle par le menu « ⋯ »
// (bas droite, visible au survol) ou le clic droit. Pas vu = vignette sans repère, vu = coche verte.
private struct EpisodeCardView: View {
    @Bindable var model: DetailModel
    let episode: Card
    let current: Bool
    let images: ImageRepository
    let onPlay: (PlayIntent) -> Void

    @State private var hovered = false

    var body: some View {
        Button(action: play) {
            HStack(alignment: .top, spacing: 20) {
                AuthenticatedImage(path: episode.thumb ?? episode.backdrop, repository: images)
                    .frame(width: 250, height: 141)
                    .clipShape(RoundedRectangle(cornerRadius: 4))
                    .overlay(alignment: .bottom) {
                        if episode.progress > 0, !episode.watched {
                            GeometryReader { proxy in
                                ZStack(alignment: .leading) {
                                    Capsule().fill(.white.opacity(0.25))
                                    Capsule().fill(AuroraTheme.accent).frame(width: proxy.size.width * episode.progress)
                                }
                            }
                            // Marge droite : laisse la place au bouton « ⋯ ».
                            .frame(height: 3).padding(.leading, 9).padding(.trailing, 42).padding(.bottom, 8)
                        }
                    }
                    .overlay(alignment: .topTrailing) {
                        if episode.watched {
                            Image(systemName: "checkmark")
                                .font(.system(size: 11, weight: .bold))
                                .foregroundStyle(.white)
                                .frame(width: 22, height: 22)
                                .background(AuroraTheme.seen, in: Circle())
                                .shadow(color: .black.opacity(0.4), radius: 2, y: 1)
                                .padding(7)
                                .accessibilityLabel("Vu")
                        }
                    }
                    .overlay {
                        if current { RoundedRectangle(cornerRadius: 5).strokeBorder(AuroraTheme.accent, lineWidth: 2).padding(-3) }
                    }
                VStack(alignment: .leading, spacing: 4) {
                    Text("Épisode \(episode.episode ?? 0)".uppercased()).font(.system(size: 10, weight: .semibold)).tracking(2).foregroundStyle(AuroraTheme.ash)
                    Text(episode.title).font(AuroraTheme.serif(16)).foregroundStyle(AuroraTheme.paper).lineLimit(2)
                    if let overview = episode.overview, !overview.isEmpty {
                        Text(overview)
                            .font(.system(size: 13)).lineSpacing(2)
                            .foregroundStyle(AuroraTheme.paper.opacity(0.75))
                            .lineLimit(3)
                            .frame(maxWidth: 620, alignment: .leading)
                            .padding(.top, 2)
                    }
                    Text([AuroraFormat.runtime(minutes: episode.runtime), episode.watched ? nil : AuroraFormat.remaining(episode)].compactMap { $0 }.joined(separator: " · "))
                        .font(.system(size: 12)).foregroundStyle(AuroraTheme.mist)
                        .padding(.top, 2)
                }
                .padding(.top, 4)
                Spacer(minLength: 0)
            }
            .padding(8)
            .background(hovered ? AuroraTheme.panel : .clear, in: RoundedRectangle(cornerRadius: 8))
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .overlay(alignment: .topLeading) {
            // Bouton « ⋯ » posé sur le coin bas droit de la vignette (250 × 141 pt, marge de ligne 8 pt).
            Menu { watchedChoices } label: {
                Image(systemName: "ellipsis")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(AuroraTheme.paper)
                    .frame(width: 26, height: 26)
                    .background(.black.opacity(0.6), in: Circle())
                    .overlay(Circle().strokeBorder(AuroraTheme.paper.opacity(0.28)))
                    .contentShape(Circle())
            }
            .menuStyle(.button)
            .buttonStyle(.plain)
            .menuIndicator(.hidden)
            .fixedSize()
            .padding(.leading, 8 + 250 - 26 - 7).padding(.top, 8 + 141 - 26 - 7)
            .opacity(hovered ? 1 : 0)
            .help("Vu ou pas vu")
            .accessibilityLabel("Options de \(episode.title)")
        }
        .onHover { hovered = $0 }
        .animation(.easeOut(duration: 0.15), value: hovered)
        .contextMenu {
            watchedChoices
            Divider()
            Button("Lire", action: play)
        }
        .help(episode.title)
    }

    /// « Vu » / « Pas vu » : l'état actuel est coché (cases à cocher natives du menu macOS).
    @ViewBuilder private var watchedChoices: some View {
        SwiftUI.Section("Épisode \(episode.episode ?? 0)") {
            Toggle("Vu", isOn: Binding(get: { episode.watched }, set: { if $0 { setWatched(true) } }))
            Toggle("Pas vu", isOn: Binding(get: { !episode.watched }, set: { if $0 { setWatched(false) } }))
        }
    }

    private func setWatched(_ watched: Bool) {
        guard watched != episode.watched else { return }
        Task { await model.setEpisodeWatched(episode, watched: watched) }
    }

    private func play() {
        onPlay(PlayIntent(itemID: episode.id, fileID: nil, startPosition: episode.watched ? 0 : episode.position ?? 0, restart: false))
    }
}

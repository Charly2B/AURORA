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

                ScrollView(.horizontal) {
                    LazyHStack(alignment: .top, spacing: 16) {
                        ForEach(model.episodes) { episode in
                            episodeCard(episode, current: episode.id == currentID)
                        }
                    }
                    .padding(.vertical, 6).padding(.horizontal, 4)
                }
                .scrollIndicators(.hidden)
                .padding(.horizontal, -4)
                .overlay { LoadingStateView(state: model.episodesState, emptyMessage: "Aucun épisode.") }
            }
        }
    }

    /// Épisode à reprendre (en cours), sinon le premier non vu : entouré d'or.
    private var currentID: Int64? {
        (model.episodes.first { ($0.position ?? 0) > 0 && !$0.watched } ?? model.episodes.first { !$0.watched })?.id
    }

    // Zone de lecture et zone « vu » séparées : un clic sur la carte lance la lecture,
    // seul le bouton ✓ bascule l'état vu.
    private func episodeCard(_ episode: Card, current: Bool) -> some View {
        Button {
            onPlay(PlayIntent(itemID: episode.id, fileID: nil, startPosition: episode.watched ? 0 : episode.position ?? 0, restart: false))
        } label: {
            VStack(alignment: .leading, spacing: 4) {
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
                            .frame(height: 3).padding(.horizontal, 9).padding(.bottom, 8)
                        }
                    }
                    .overlay {
                        if current { RoundedRectangle(cornerRadius: 5).strokeBorder(AuroraTheme.accent, lineWidth: 2).padding(-3) }
                    }
                    .padding(.bottom, 6)
                Text("Épisode \(episode.episode ?? 0)".uppercased()).font(.system(size: 10, weight: .semibold)).tracking(2).foregroundStyle(AuroraTheme.ash)
                Text(episode.title).font(AuroraTheme.serif(15)).foregroundStyle(AuroraTheme.paper).lineLimit(2)
                Text([AuroraFormat.runtime(minutes: episode.runtime), episode.watched ? nil : AuroraFormat.remaining(episode)].compactMap { $0 }.joined(separator: " · "))
                    .font(.system(size: 12)).foregroundStyle(AuroraTheme.mist)
            }
            .frame(width: 250, alignment: .leading)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .overlay(alignment: .topTrailing) {
            Button {
                Task { await model.setEpisodeWatched(episode, watched: !episode.watched) }
            } label: {
                Image(systemName: "checkmark")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(episode.watched ? AuroraTheme.ink : AuroraTheme.paper.opacity(0.8))
                    .frame(width: 24, height: 24)
                    .background(episode.watched ? AnyShapeStyle(AuroraTheme.accent) : AnyShapeStyle(.black.opacity(0.45)), in: Circle())
                    .padding(7)
            }
            .buttonStyle(.plain)
            .help(episode.watched ? "Marquer comme non vu" : "Marquer comme vu")
            .accessibilityLabel(episode.watched ? "Marquer \(episode.title) comme non vu" : "Marquer \(episode.title) comme vu")
        }
        .contextMenu {
            Button {
                Task { await model.setEpisodeWatched(episode, watched: !episode.watched) }
            } label: {
                Label(episode.watched ? "Marquer comme non vu" : "Marquer comme vu", systemImage: episode.watched ? "eye.slash" : "eye")
            }
            Divider()
            Button("Lire") {
                onPlay(PlayIntent(itemID: episode.id, fileID: nil, startPosition: episode.watched ? 0 : episode.position ?? 0, restart: false))
            }
        }
        .help(episode.overview ?? episode.title)
    }
}

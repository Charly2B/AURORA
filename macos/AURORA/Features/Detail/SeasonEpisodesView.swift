import SwiftUI

struct SeasonEpisodesView: View {
    @Bindable var model: DetailModel
    let images: ImageRepository
    let onPlay: (PlayIntent) -> Void

    var body: some View {
        if let seasons = model.detail?.seasons, !seasons.isEmpty {
            VStack(alignment: .leading, spacing: 16) {
                ScrollView(.horizontal) {
                    HStack(spacing: 10) {
                        ForEach(seasons) { season in
                            Button(season.title) { Task { await model.selectSeason(season.number) } }
                                .buttonStyle(AuroraChipStyle(selected: model.selectedSeason == season.number))
                        }
                    }
                }.scrollIndicators(.hidden)

                LazyVStack(spacing: 10) {
                    ForEach(model.episodes) { episode in
                        // Zone de lecture et zone « vu » séparées : un clic sur la ligne
                        // lance la lecture, seul le bouton ✓ bascule l'état vu.
                        HStack(spacing: 12) {
                            Button {
                                onPlay(PlayIntent(itemID: episode.id, fileID: nil, startPosition: episode.watched ? 0 : episode.position ?? 0, restart: false))
                            } label: {
                                HStack(spacing: 18) {
                                    AuthenticatedImage(path: episode.thumb, repository: images)
                                        .frame(width: 220, height: 124)
                                        .clipShape(RoundedRectangle(cornerRadius: 8))
                                    VStack(alignment: .leading, spacing: 7) {
                                        Text(Self.episodeCode(episode)).font(.caption.bold()).foregroundStyle(AuroraTheme.ash)
                                        Text(episode.title).font(.title3.weight(.semibold))
                                        Text(episode.overview ?? "").lineLimit(2).foregroundStyle(AuroraTheme.mist)
                                    }
                                    Spacer()
                                }
                                .contentShape(Rectangle())
                            }
                            .buttonStyle(.plain)

                            Button {
                                Task { await model.setEpisodeWatched(episode, watched: !episode.watched) }
                            } label: {
                                Image(systemName: episode.watched ? "checkmark.circle.fill" : "checkmark.circle")
                                    .font(.title2)
                                    .foregroundStyle(episode.watched ? AuroraTheme.accent : AuroraTheme.ash)
                                    .frame(width: 36, height: 36)
                            }
                            .buttonStyle(.plain)
                            .help(episode.watched ? "Marquer comme non vu" : "Marquer comme vu")
                            .accessibilityLabel(episode.watched ? "Marquer \(episode.title) comme non vu" : "Marquer \(episode.title) comme vu")
                        }
                        .padding(10)
                        .background(AuroraTheme.panel, in: RoundedRectangle(cornerRadius: 12))
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
                    }
                }
                .overlay { LoadingStateView(state: model.episodesState, emptyMessage: "Aucun épisode.") }
            }
        }
    }

    private static func episodeCode(_ card: Card) -> String {
        String(format: "S%02dE%02d", card.season ?? 0, card.episode ?? 0)
    }
}

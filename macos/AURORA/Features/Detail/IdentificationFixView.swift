import SwiftUI

// Correction d'identification : rechercher la bonne fiche TMDB et l'associer
// (administrateur). Après association, affiche et fond deviennent modifiables.
struct IdentificationFixView: View {
    @Bindable var model: DetailModel
    let onClose: () -> Void

    @State private var query = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(spacing: 14) {
                Text("Corriger l'identification").font(.title2.bold())
                if let title = model.detail?.title {
                    Text(title).font(.callout).foregroundStyle(AuroraTheme.mist).lineLimit(1)
                }
                Spacer()
                Button("Fermer") { onClose() }.buttonStyle(.bordered)
            }

            HStack(spacing: 10) {
                Image(systemName: "magnifyingglass").foregroundStyle(AuroraTheme.ash)
                TextField("Titre à rechercher sur TMDB", text: $query)
                    .textFieldStyle(.plain)
                    .font(.body)
                    .onSubmit { Task { await model.loadCandidates(query: query) } }
            }
            .padding(10)
            .background(AuroraTheme.panel, in: RoundedRectangle(cornerRadius: 10))

            ScrollView {
                LazyVStack(spacing: 8) {
                    ForEach(model.candidates) { candidate in
                        Button {
                            Task { await model.applyMatch(candidate.id) }
                        } label: {
                            HStack(alignment: .top, spacing: 14) {
                                AsyncImage(url: URL(string: candidate.poster ?? "")) { image in
                                    image.resizable().aspectRatio(contentMode: .fill)
                                } placeholder: {
                                    AuroraTheme.panel
                                }
                                .frame(width: 52, height: 78)
                                .clipShape(RoundedRectangle(cornerRadius: 6))
                                VStack(alignment: .leading, spacing: 5) {
                                    HStack(spacing: 8) {
                                        Text(candidate.title).font(.headline)
                                        if let year = candidate.year, year > 0 {
                                            Text(String(year)).font(.callout).foregroundStyle(AuroraTheme.mist)
                                        }
                                    }
                                    if let original = candidate.originalTitle, !original.isEmpty, original != candidate.title {
                                        Text(original).font(.caption).foregroundStyle(AuroraTheme.ash)
                                    }
                                    if let overview = candidate.overview, !overview.isEmpty {
                                        Text(overview).font(.caption).foregroundStyle(AuroraTheme.mist).lineLimit(2)
                                    }
                                }
                                Spacer()
                                Image(systemName: "checkmark.circle")
                                    .foregroundStyle(AuroraTheme.ash)
                            }
                            .padding(10)
                            .background(AuroraTheme.panel, in: RoundedRectangle(cornerRadius: 10))
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        .help("Associer cette fiche TMDB")
                    }
                }
                .padding(.vertical, 4)
            }
            .overlay { LoadingStateView(state: model.candidatesState, emptyMessage: "Aucun résultat TMDB.") }

            if let error = model.errorMessage { Text(error).foregroundStyle(.red) }
        }
        .padding(24)
        .frame(width: 780, height: 640)
        .background(AuroraTheme.background)
        .foregroundStyle(AuroraTheme.paper)
        .task(id: query) {
            // Recherche à la frappe avec un court délai, comme sur l'interface web.
            try? await Task.sleep(for: .milliseconds(400))
            guard !Task.isCancelled else { return }
            await model.loadCandidates(query: query)
        }
    }
}
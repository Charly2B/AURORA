import SwiftUI

struct LibraryView: View {
    @Bindable var model: LibraryModel
    let images: ImageRepository
    let onOpen: (Card) -> Void
    let memory: BrowseMemory
    let origin: String
    let tab: BrowseTab
    @State private var grouped = false
    @State private var scrollTarget: Int64?
    @State private var openedItemID: Int64?

    private let columns = [GridItem(.adaptive(minimum: 150, maximum: 220), spacing: 20)]

    var body: some View {
        VStack(spacing: 0) {
            ScrollView(.horizontal) {
                HStack(spacing: 10) {
                    ForEach(model.libraries) { library in
                        Button(library.name) {
                            grouped = false
                            model.selectedLibraryID = library.id
                            saveMemory()
                            Task { await model.loadSelected() }
                        }.buttonStyle(AuroraChipStyle(selected: !grouped && model.selectedLibraryID == library.id))
                    }
                    if model.libraries.count > 1 {
                        Button("Tous les dossiers") { grouped = true; saveMemory(); Task { await model.loadGrouped() } }
                            .buttonStyle(AuroraChipStyle(selected: grouped))
                    }
                    Divider().frame(height: 26)
                    ForEach(LibrarySort.available(for: model.kind), id: \.self) { sort in
                        Button(sort.label) {
                            model.sort = sort
                            saveMemory()
                            Task { grouped ? await model.loadGrouped() : await model.loadSelected() }
                        }.buttonStyle(AuroraChipStyle(selected: model.sort == sort))
                    }
                }.padding(.horizontal, 36).padding(.vertical, 12)
            }.scrollIndicators(.hidden)

            ScrollView {
                if grouped {
                    LazyVStack(spacing: 28) {
                        ForEach(model.folders.filter { !$0.items.isEmpty }) { folder in
                            FolderPanelView(folder: folder, kind: model.kind, images: images, onOpen: open)
                                .id(folder.id)
                        }
                    }.scrollTargetLayout().padding(36)
                } else {
                    LazyVGrid(columns: columns, spacing: 22) {
                        ForEach(model.items) { card in
                            MediaCardView(card: card, images: images, onOpen: { open(card) })
                                .id(card.id)
                        }
                    }.scrollTargetLayout().padding(36)
                }
            }
            .scrollPosition(id: $scrollTarget, anchor: .center)
            .overlay { LoadingStateView(state: model.state, emptyMessage: model.kind.emptyMessage) }
        }
        .task {
            let saved = memory.restore(rootKey)
            if let saved {
                model.sort = saved.sort
                model.selectedLibraryID = saved.selectedLibraryID
                grouped = saved.grouped
                openedItemID = saved.openedItemID
            }
            if grouped { await model.loadGrouped() } else { await model.loadSelected() }
            scrollTarget = saved?.scrollTarget ?? saved?.openedItemID
        }
        .onChange(of: scrollTarget) { _, _ in saveMemory() }
        .onDisappear { saveMemory() }
    }

    private var rootKey: BrowseKey { BrowseKey(origin: origin, tab: tab, libraryID: nil) }

    private func open(_ card: Card) {
        openedItemID = card.id
        scrollTarget = card.id
        saveMemory()
        onOpen(card)
    }

    private func saveMemory() {
        let state = BrowseState(
            sort: model.sort,
            grouped: grouped,
            selectedLibraryID: model.selectedLibraryID,
            scrollTarget: scrollTarget,
            openedItemID: openedItemID
        )
        memory.save(state, for: rootKey)
        memory.save(state, for: BrowseKey(origin: origin, tab: tab, libraryID: model.selectedLibraryID))
    }
}

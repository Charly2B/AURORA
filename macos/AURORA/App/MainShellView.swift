import SwiftUI

struct MainShellView: View {
    @Bindable var session: AppSession
    let api: any AuroraServing
    let server: ServerAddress
    let profile: UserProfile

    private let images: ImageRepository
    private let capabilities: Capabilities
    @State private var selectedTab: MainTab = .home
    @State private var path: [AppRoute] = []
    @State private var homeModel: HomeModel
    @State private var moviesModel: LibraryModel
    @State private var seriesModel: LibraryModel
    @State private var searchModel: SearchModel
    @State private var profileModel: ProfileModel
    @State private var catalogGeneration = 0
    @State private var browseMemory = BrowseMemory()

    init(session: AppSession, api: any AuroraServing, server: ServerAddress, profile: UserProfile) {
        self.session = session
        self.api = api
        self.server = server
        self.profile = profile
        let images = ImageRepository(api: api)
        let capabilities = PlaybackCapabilities.detect()
        self.images = images
        self.capabilities = capabilities
        _homeModel = State(initialValue: HomeModel(api: api, origin: server.origin))
        _moviesModel = State(initialValue: LibraryModel(api: api, origin: server.origin, kind: .movie))
        _seriesModel = State(initialValue: LibraryModel(api: api, origin: server.origin, kind: .series))
        _searchModel = State(initialValue: SearchModel(api: api, origin: server.origin))
        _profileModel = State(initialValue: ProfileModel(user: profile, server: server, api: api, capabilities: capabilities))
    }

    var body: some View {
        NavigationStack(path: $path) {
            HStack(spacing: 0) {
                sidebar
                Divider()
                tabContent
            }
            .background(AuroraTheme.background)
            .foregroundStyle(AuroraTheme.paper)
            .navigationDestination(for: AppRoute.self) { route in
                destination(route).id(route.identity)
            }
        }
    }

    private var sidebar: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("AURORA").font(.title2.bold()).padding(.horizontal, 12).padding(.bottom, 20)
            ForEach(MainTab.allCases) { tab in
                Button {
                    path.removeAll()
                    selectedTab = tab
                } label: {
                    Label(tab.label, systemImage: tab.symbol)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 12).padding(.vertical, 9)
                        .background(selectedTab == tab ? AuroraTheme.elevated : .clear, in: RoundedRectangle(cornerRadius: 9))
                }
                .buttonStyle(.plain)
                .accessibilityLabel(tab.label)
            }
            Spacer()
            Text(server.url.host ?? server.origin).font(.caption).foregroundStyle(AuroraTheme.ash).padding(12)
        }
        .padding(12)
        .frame(width: 190)
        .background(Color(white: 0.035))
    }

    @ViewBuilder
    private var tabContent: some View {
        switch selectedTab {
        case .home:
            HomeView(model: homeModel, images: images, onOpen: open, onPlay: play, onHideContinue: hideContinue)
        case .movies:
            LibraryView(model: moviesModel, images: images, onOpen: open, memory: browseMemory, origin: server.origin, tab: .movies)
        case .series:
            LibraryView(model: seriesModel, images: images, onOpen: open, memory: browseMemory, origin: server.origin, tab: .series)
        case .search:
            SearchView(model: searchModel, images: images, onOpen: open)
        case .profile:
            ProfileView(
                model: profileModel,
                onSwitchServer: session.showServers,
                onAddServer: { session.phase = .addServer },
                onLogout: { Task { await session.logout() } }
            )
        }
    }

    @ViewBuilder
    private func destination(_ route: AppRoute) -> some View {
        switch route {
        case .home:
            EmptyView()
        case .detail(let itemID):
            DetailDestination(itemID: itemID, api: api, images: images, catalogGeneration: catalogGeneration, onChanged: refreshCatalogs, onOpen: open, onPlay: { intent, title in
                path.append(.player(intent, title: title))
            })
        case .player(let intent, let title):
            PlayerDestination(
                intent: intent,
                title: title,
                api: api,
                server: server,
                profile: profile,
                capabilities: capabilities,
                images: images,
                onClose: {
                    refreshCatalogs()
                    if !path.isEmpty { path.removeLast() }
                }
            )
        }
    }

    private func open(_ card: Card) { path.append(.detail(itemID: card.id)) }

    private func play(_ card: Card) {
        path.append(.player(
            PlayIntent(itemID: card.id, fileID: nil, startPosition: card.position ?? 0, restart: false),
            title: card.title
        ))
    }

    private func hideContinue(_ card: Card) {
        Task {
            try? await api.hideContinue(itemID: card.id)
            await homeModel.load()
        }
    }

    private func refreshCatalogs() {
        catalogGeneration += 1
        Task {
            async let home: Void = homeModel.load()
            async let movies: Void = moviesModel.reload()
            async let series: Void = seriesModel.reload()
            _ = await (home, movies, series)
        }
    }
}

private struct DetailDestination: View {
    @State private var model: DetailModel
    let images: ImageRepository
    let catalogGeneration: Int
    let onOpen: (Card) -> Void
    let onPlay: (PlayIntent, String) -> Void
    @Environment(\.dismiss) private var dismiss

    init(itemID: Int64, api: any AuroraServing, images: ImageRepository, catalogGeneration: Int, onChanged: @escaping @MainActor () -> Void, onOpen: @escaping (Card) -> Void, onPlay: @escaping (PlayIntent, String) -> Void) {
        _model = State(initialValue: DetailModel(itemID: itemID, api: api, onChanged: onChanged))
        self.images = images
        self.catalogGeneration = catalogGeneration
        self.onOpen = onOpen
        self.onPlay = onPlay
    }

    var body: some View {
        DetailView(model: model, images: images, onBack: { dismiss() }, onOpen: onOpen) { intent in
            onPlay(intent, model.detail?.title ?? "Lecture")
        }
        .navigationBarBackButtonHidden()
        .onChange(of: catalogGeneration) { _, _ in Task { await model.load() } }
    }
}

private struct PlayerDestination: View {
    @State private var coordinator: PlaybackCoordinator
    private let controller: LibMPVController
    let intent: PlayIntent
    let title: String
    let images: ImageRepository
    let onClose: () -> Void

    init(
        intent: PlayIntent,
        title: String,
        api: any AuroraServing,
        server: ServerAddress,
        profile: UserProfile,
        capabilities: Capabilities,
        images: ImageRepository,
        onClose: @escaping () -> Void
    ) {
        let controller = LibMPVController()
        self.controller = controller
        _coordinator = State(initialValue: PlaybackCoordinator(
            api: api,
            mpv: controller,
            sessions: KeychainSessionStore(),
            server: server,
            capabilities: capabilities,
            preferences: profile.prefs
        ))
        self.intent = intent
        self.title = title
        self.images = images
        self.onClose = onClose
    }

    var body: some View {
        PlayerView(coordinator: coordinator, controller: controller, intent: intent, title: title, images: images, onClose: onClose)
            .navigationBarBackButtonHidden()
    }
}

import AppKit
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
    @AppStorage(HomeBackgroundStore.pathKey) private var customBackgroundPath = ""

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
            ZStack {
                shellBackground

                HStack(spacing: 0) {
                    sidebar.zIndex(1)
                    tabContent
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .foregroundStyle(AuroraTheme.paper)
            .navigationDestination(for: AppRoute.self) { route in
                destination(route).id(route.identity)
            }
        }
        // Événements serveur (SSE, administrateur) : dès qu'une analyse se termine —
        // p. ex. un film supprimé dans le Finder — les listes et fiches ouvertes se rechargent.
        .task {
            guard profile.role == "admin" else { return }
            let events = await api.scanEvents()
            for await update in events {
                if update.isTerminal { refreshCatalogs() }
            }
        }
    }

    // Colonne « Cinémathèque » : Recherche en champ (⌘F), Parcourir avec compteurs,
    // Profil + serveur en pied. L'ordre des onglets reste celui de MainTab.
    private var sidebar: some View {
        VStack(alignment: .leading, spacing: 0) {
            VStack(alignment: .leading, spacing: 0) {
                Text("MEDIATHEQUE\nAURORA")
                    .font(.system(size: 18, weight: .light))
                    .tracking(2.8)
                    .lineSpacing(3)
                    .foregroundStyle(AuroraTheme.paper)
                Rectangle().fill(AuroraTheme.accent).frame(width: 30, height: 1).padding(.vertical, 8)
            }
            .padding(.leading, 10).padding(.top, 14).padding(.bottom, 20)

            Button { select(.search) } label: {
                HStack(spacing: 8) {
                    Image(systemName: "magnifyingglass")
                    Text("Rechercher")
                    Spacer()
                    Text("⌘F").font(.system(size: 11, weight: .medium)).opacity(0.75)
                }
                .font(.system(size: 13))
                .foregroundStyle(selectedTab == .search ? AuroraTheme.paper : AuroraTheme.ash)
                .padding(.horizontal, 10).frame(height: 32)
                .background(AuroraTheme.panel, in: RoundedRectangle(cornerRadius: 6))
                .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(selectedTab == .search ? AuroraTheme.accent.opacity(0.6) : AuroraTheme.line))
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .keyboardShortcut("f", modifiers: .command)
            .accessibilityLabel("Rechercher")
            .padding(.bottom, 20)

            AuroraEyebrow(text: "Parcourir", color: AuroraTheme.ash).padding(.leading, 10).padding(.bottom, 7)
            navItem(.home, count: nil)
            navItem(.movies, count: homeModel.movieCount)
            navItem(.series, count: homeModel.seriesCount)

            Spacer(minLength: 12)

            Divider().overlay(AuroraTheme.line)
            Button { select(.profile) } label: {
                HStack(spacing: 10) {
                    Image(systemName: "person")
                        .frame(width: 32, height: 32)
                        .foregroundStyle(AuroraTheme.accent)
                        .overlay(Circle().strokeBorder(AuroraTheme.accent.opacity(0.5)))
                    VStack(alignment: .leading, spacing: 1) {
                        Text("Profil").font(AuroraTheme.serif(14)).foregroundStyle(AuroraTheme.paper)
                        HStack(spacing: 6) {
                            Circle().fill(AuroraTheme.green).frame(width: 6, height: 6)
                            Text(server.url.host ?? server.origin).lineLimit(1)
                        }
                        .font(.system(size: 11)).foregroundStyle(AuroraTheme.ash)
                    }
                    Spacer(minLength: 0)
                    Image(systemName: "gearshape").foregroundStyle(selectedTab == .profile ? AuroraTheme.accent : AuroraTheme.ash)
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .padding(.top, 12)
            .accessibilityLabel("Profil et réglages")
        }
        .padding(14)
        .frame(width: 280, alignment: .leading)
        .clipped()
        .background {
            if showsLandingBackground {
                Color.black.opacity(0.58)
            } else {
                AuroraTheme.sidebar
            }
        }
    }

    @ViewBuilder
    private var shellBackground: some View {
        if showsLandingBackground {
            Group {
                if !customBackgroundPath.isEmpty {
                    HomeBackgroundView(path: customBackgroundPath)
                } else {
                    GeometryReader { geometry in
                        Image("MediathequeAuroraBackdrop")
                            .resizable()
                            .scaledToFill()
                            .frame(width: geometry.size.width, height: geometry.size.height)
                            .clipped()
                            .overlay(Color.black.opacity(0.12))
                    }
                    .ignoresSafeArea()
                }
            }
            .allowsHitTesting(false)
        } else {
            AuroraTheme.background
                .ignoresSafeArea()
        }
    }

    private var showsLandingBackground: Bool {
        selectedTab == .home && path.isEmpty
    }

    private func select(_ tab: MainTab) {
        path.removeAll()
        selectedTab = tab
    }

    private func navItem(_ tab: MainTab, count: Int?) -> some View {
        let selected = selectedTab == tab && path.isEmpty
        return Button { select(tab) } label: {
            HStack(spacing: 11) {
                Image(systemName: tab.symbol).font(.system(size: 14)).frame(width: 18)
                    .foregroundStyle(selected ? AuroraTheme.accent : AuroraTheme.ash)
                Text(tab.label).font(AuroraTheme.serif(16))
                Spacer()
                if let count, count > 0 {
                    Text("\(count)").font(.system(size: 12)).monospacedDigit().foregroundStyle(AuroraTheme.ash)
                }
            }
            .foregroundStyle(selected ? AuroraTheme.paper : AuroraTheme.mist)
            .padding(.horizontal, 10).frame(height: 36)
            .background {
                if selected {
                    LinearGradient(colors: [AuroraTheme.accent.opacity(0.17), AuroraTheme.accent.opacity(0.03)], startPoint: .leading, endPoint: .trailing)
                        .overlay(alignment: .leading) { Rectangle().fill(AuroraTheme.accent).frame(width: 2) }
                        .clipShape(RoundedRectangle(cornerRadius: 6))
                }
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .padding(.bottom, 2)
        .accessibilityLabel(tab.label)
    }

    @ViewBuilder
    private var tabContent: some View {
        switch selectedTab {
        case .home:
            HomeView(model: homeModel)
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
            DetailDestination(itemID: itemID, api: api, images: images, canEditArtwork: profile.role == "admin", catalogGeneration: catalogGeneration, onChanged: refreshCatalogs, onOpen: open, onPlay: { intent, title in
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

    init(itemID: Int64, api: any AuroraServing, images: ImageRepository, canEditArtwork: Bool, catalogGeneration: Int, onChanged: @escaping @MainActor () -> Void, onOpen: @escaping (Card) -> Void, onPlay: @escaping (PlayIntent, String) -> Void) {
        _model = State(initialValue: DetailModel(itemID: itemID, api: api, canEditArtwork: canEditArtwork, onChanged: onChanged))
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

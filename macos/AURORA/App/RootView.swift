import SwiftUI

struct RootView: View {
    @State private var session: AppSession

    init(session: AppSession = AppSession()) {
        _session = State(initialValue: session)
    }

    var body: some View {
        Group {
            switch session.phase {
            case .loading:
                ZStack { Color.black; ProgressView("Connexion à AURORA…") }
            case .serverList:
                ServerListView(
                    servers: session.servers,
                    onSelect: { server in Task { await session.select(server) } },
                    onDelete: { server in Task { try? await session.remove(server) } },
                    onAdd: { session.phase = .addServer }
                )
            case .addServer:
                SetupContainer(session: session, mode: .add)
            case .login:
                SetupContainer(session: session, mode: .login)
            case .authenticated:
                if let api = session.service(), let server = session.selectedServer, let profile = session.profile {
                    MainShellView(session: session, api: api, server: server, profile: profile)
                        .id(session.shellIdentity)
                } else {
                    ProgressView()
                }
            }
        }
        .frame(minWidth: 960, minHeight: 640)
        .preferredColorScheme(.dark)
        .task { if session.phase == .loading { await session.restore() } }
    }
}

private struct SetupContainer: View {
    enum Mode { case add, login }
    @Bindable var session: AppSession
    @State private var model: SetupModel
    let mode: Mode

    init(session: AppSession, mode: Mode) {
        self.session = session
        self.mode = mode
        _model = State(initialValue: SetupModel(session: session))
    }

    var body: some View {
        switch mode {
        case .add:
            AddServerView(model: model, onBack: session.showServers)
        case .login:
            if let server = session.selectedServer {
                LoginView(model: model, server: server, onBack: session.showServers)
            } else {
                ServerListView(servers: session.servers, onSelect: { server in Task { await session.select(server) } }, onDelete: { _ in }, onAdd: { session.phase = .addServer })
            }
        }
    }
}


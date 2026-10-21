import SwiftUI

struct LoginView: View {
    @Bindable var model: SetupModel
    let server: ServerAddress
    let onBack: () -> Void

    @State private var pollingTask: Task<Void, Never>?

    var body: some View {
        VStack(spacing: 28) {
            HStack {
                Button("Retour", action: onBack)
                Spacer()
                Text(server.origin).foregroundStyle(.secondary)
            }

            Text("Connexion AURORA")
                .font(.largeTitle.bold())

            if let link = model.link {
                LinkCodeView(link: link, server: server, state: model.state) {
                    pollingTask?.cancel()
                    pollingTask = Task {
                        await model.restartLink()
                        await model.pollLink()
                    }
                }
            } else {
                ProgressView("Création du code…")
            }

            Divider().frame(maxWidth: 440)
            Text("Ou utilisez votre mot de passe")
                .font(.headline)
            Text("Compte AURORA — pas le compte Synology")
                .font(.subheadline)
                .foregroundStyle(.secondary)
            TextField("Utilisateur", text: $model.username)
                .textFieldStyle(.roundedBorder)
                .frame(maxWidth: 380)
                .textContentType(.username)
            SecureField("Mot de passe", text: $model.password)
                .textFieldStyle(.roundedBorder)
                .frame(maxWidth: 380)
                .textContentType(.password)
                .onSubmit { submitPassword() }
            Button {
                submitPassword()
            } label: {
                if model.isAuthenticating {
                    ProgressView().controlSize(.small)
                } else {
                    Text("Se connecter")
                }
            }
                .buttonStyle(.borderedProminent)
                .disabled(
                    model.isAuthenticating ||
                    model.username.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
                    model.password.isEmpty
                )
            if let error = model.errorMessage {
                Text(error).foregroundStyle(.red)
            }
        }
        .padding(36)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.black)
        .foregroundStyle(.white)
        .task {
            do {
                try await model.beginLink()
                pollingTask = Task { await model.pollLink() }
            } catch {}
        }
        .onDisappear {
            pollingTask?.cancel()
            model.cancelPolling()
        }
    }

    private func submitPassword() {
        let username = model.username
        let password = model.password
        Task { await model.login(username: username, password: password) }
    }
}

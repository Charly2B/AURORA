import SwiftUI

struct ProfileView: View {
    @Bindable var model: ProfileModel
    let onSwitchServer: () -> Void
    let onAddServer: () -> Void
    let onLogout: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Image(systemName: "person.crop.circle.fill")
                .font(.system(size: 72)).foregroundStyle(AuroraTheme.mist)
                .accessibilityHidden(true)
            Text(model.username).font(.system(size: 40, weight: .bold))
            Text(model.serverAddress).foregroundStyle(AuroraTheme.mist).textSelection(.enabled)
            LabeledContent("Serveur AURORA", value: model.serverVersion)
            Text("Lecture : \(model.playbackSummary)").foregroundStyle(AuroraTheme.ash)
            if let error = model.errorMessage { Text(error).foregroundStyle(.red) }
            HStack(spacing: 12) {
                Button("Changer de NAS", action: onSwitchServer)
                Button("Ajouter un NAS", action: onAddServer)
                Button("Déconnexion", role: .destructive, action: onLogout)
            }.buttonStyle(.bordered)
            Spacer()
        }
        .padding(44)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .task { await model.load() }
    }
}

import SwiftUI

struct AddServerView: View {
    @Bindable var model: SetupModel
    let onBack: () -> Void

    var body: some View {
        VStack(spacing: 22) {
            Text("Ajouter un NAS")
                .font(.largeTitle.bold())
            TextField("nas.local ou 192.168.1.20", text: $model.serverAddress)
                .textFieldStyle(.roundedBorder)
                .frame(maxWidth: 440)
                .onSubmit { Task { await model.addServer() } }
                .accessibilityLabel("Adresse du NAS")
            if let error = model.errorMessage {
                Text(error).foregroundStyle(.red)
            }
            HStack {
                Button("Retour", action: onBack)
                Button("Connexion") { Task { await model.addServer() } }
                    .buttonStyle(.borderedProminent)
                    .disabled(model.serverAddress.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(48)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.black)
        .foregroundStyle(.white)
    }
}

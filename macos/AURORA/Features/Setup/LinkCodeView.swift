import AppKit
import SwiftUI

struct LinkCodeView: View {
    let link: LinkStart
    let server: ServerAddress
    let state: SetupState
    let onRestart: () -> Void

    var body: some View {
        VStack(spacing: 16) {
            Text("Code de connexion")
                .font(.headline)
            Text(link.code)
                .font(.system(size: 42, weight: .bold, design: .monospaced))
                .textSelection(.enabled)
                .accessibilityLabel("Code \(link.code)")
            Text(state == .expired ? "Ce code a expiré." : "Validez ce code depuis AURORA sur votre NAS.")
                .foregroundStyle(state == .expired ? .red : .secondary)
            HStack {
                Button("Ouvrir AURORA dans le navigateur") {
                    NSWorkspace.shared.open(server.linkURL)
                }
                if state == .expired {
                    Button("Nouveau code", action: onRestart)
                        .buttonStyle(.borderedProminent)
                }
            }
        }
    }
}

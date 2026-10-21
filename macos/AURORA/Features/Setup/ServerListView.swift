import SwiftUI

struct ServerListView: View {
    let servers: [ServerAddress]
    let onSelect: (ServerAddress) -> Void
    let onDelete: (ServerAddress) -> Void
    let onAdd: () -> Void

    var body: some View {
        VStack(spacing: 28) {
            Text("AURORA")
                .font(.system(size: 34, weight: .bold, design: .rounded))
            Text("Choisissez votre serveur multimédia")
                .foregroundStyle(.secondary)

            VStack(spacing: 10) {
                ForEach(servers) { server in
                    HStack {
                        Button(server.origin) { onSelect(server) }
                            .buttonStyle(.plain)
                            .frame(maxWidth: .infinity, alignment: .leading)
                        Button(role: .destructive) { onDelete(server) } label: {
                            Image(systemName: "trash")
                        }
                        .buttonStyle(.borderless)
                        .accessibilityLabel("Supprimer \(server.origin)")
                    }
                    .padding(14)
                    .background(.white.opacity(0.07), in: RoundedRectangle(cornerRadius: 12))
                }
            }
            .frame(maxWidth: 520)

            Button("Ajouter un NAS", action: onAdd)
                .buttonStyle(.borderedProminent)
                .controlSize(.large)
        }
        .padding(48)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.black)
        .foregroundStyle(.white)
    }
}

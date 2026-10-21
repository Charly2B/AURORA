import Foundation

struct ServerAddress: Codable, Hashable, Identifiable, Sendable {
    enum ValidationError: LocalizedError {
        case empty
        case invalid
        case unsupportedScheme
        case unsafeComponents

        var errorDescription: String? {
            switch self {
            case .empty: "Saisissez l’adresse du NAS."
            case .invalid: "Cette adresse AURORA n’est pas valide."
            case .unsupportedScheme: "Utilisez une adresse HTTP ou HTTPS."
            case .unsafeComponents: "Saisissez uniquement l’adresse du serveur, sans chemin ni identifiants."
            }
        }
    }

    let url: URL

    var id: String { origin }
    var origin: String { url.absoluteString }
    var linkURL: URL { url.appending(path: "link") }

    static func normalize(_ input: String) throws -> ServerAddress {
        let trimmed = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { throw ValidationError.empty }

        let candidate: String
        if trimmed.contains("://") {
            candidate = trimmed
        } else if trimmed.hasPrefix("[") {
            candidate = "http://\(trimmed)"
        } else if trimmed.filter({ $0 == ":" }).count >= 2 {
            candidate = "http://[\(trimmed.replacingOccurrences(of: "%", with: "%25"))]"
        } else {
            candidate = "http://\(trimmed)"
        }

        guard var components = URLComponents(string: candidate),
              let scheme = components.scheme?.lowercased(),
              let host = components.host,
              !host.isEmpty else {
            throw ValidationError.invalid
        }
        guard scheme == "http" || scheme == "https" else {
            throw ValidationError.unsupportedScheme
        }
        guard components.user == nil,
              components.password == nil,
              components.query == nil,
              components.fragment == nil,
              components.path.isEmpty || components.path == "/" else {
            throw ValidationError.unsafeComponents
        }

        components.scheme = scheme
        components.host = host.lowercased()
        components.port = components.port ?? 8890
        components.path = ""
        guard let normalizedURL = components.url else { throw ValidationError.invalid }
        return ServerAddress(url: normalizedURL)
    }
}


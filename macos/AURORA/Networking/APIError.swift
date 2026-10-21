import Foundation

struct APIError: LocalizedError, Sendable, Equatable {
    let code: Int
    let message: String

    var errorDescription: String? { message }

    static func network(_ error: URLError) -> APIError {
        if error.code == .networkConnectionLost {
            return APIError(code: 0, message: "Connexion au NAS interrompue (réseau).")
        }
        return APIError(code: 0, message: "Le serveur AURORA ne répond pas (réseau). Vérifiez l’adresse et que le NAS est allumé.")
    }

    static func unexpected(server: ServerAddress) -> APIError {
        APIError(code: 0, message: "Réponse inattendue : est-ce bien AURORA à l’adresse \(server.origin) ?")
    }
}


import Testing
@testable import AURORA

@Suite("Normalisation des adresses NAS")
struct ServerAddressTests {
    @Test(arguments: [
        ("nas", "http://nas:8890"),
        ("192.168.1.37", "http://192.168.1.37:8890"),
        ("https://nas.local:9443", "https://nas.local:9443"),
        ("fe80::1", "http://[fe80::1]:8890"),
        ("[fe80::1]:9000", "http://[fe80::1]:9000"),
        ("  http://nas:8890/  ", "http://nas:8890")
    ])
    func normalizesSupportedInputs(input: String, expected: String) throws {
        #expect(try ServerAddress.normalize(input).origin == expected)
    }

    @Test func normalizedOriginsAreStableDuplicateIdentifiers() throws {
        let short = try ServerAddress.normalize("nas")
        let complete = try ServerAddress.normalize("http://nas:8890/")
        #expect(short.id == complete.id)
        #expect(short == complete)
    }

    @Test func linkApprovalURLTargetsTheNASLinkPage() throws {
        let server = try ServerAddress.normalize("192.168.1.37")
        #expect(server.linkURL.absoluteString == "http://192.168.1.37:8890/link")
    }

    @Test(arguments: [
        "",
        "http://nas:8890/films",
        "http://user:secret@nas:8890",
        "http://nas:8890?q=secret",
        "ftp://nas"
    ])
    func rejectsUnsafeOrAmbiguousInputs(input: String) {
        #expect(throws: (any Error).self) {
            try ServerAddress.normalize(input)
        }
    }
}


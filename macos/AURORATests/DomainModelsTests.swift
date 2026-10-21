import Foundation
import Testing
@testable import AURORA

private final class FixtureMarker {}

@Suite("Modèles API AURORA")
struct DomainModelsTests {
    @Test func decodesStatusWithServerDefaultsAndUnknownFields() throws {
        let status = try JSONDecoder().decode(Status.self, from: fixture("status"))

        #expect(status.version == "0.1.8")
        #expect(status.hasAdmin == false)
        #expect(status.setupDone)
        #expect(status.user?.username == "remy")
        #expect(status.user?.prefs.subLang == "")
    }

    @Test func decodesHomeCardsAndDerivesProgress() throws {
        let home = try JSONDecoder().decode(Home.self, from: fixture("home"))

        #expect(home.hero?.id == 42)
        #expect(home.hero?.favorite == true)
        #expect(home.sections.map(\.title) == ["Continuer à regarder"])
        #expect(home.sections[0].items[0].seriesTitle == "AURORA")
        #expect(home.sections[0].items[0].progress == 120.0 / 3600.0)
    }

    private func fixture(_ name: String) throws -> Data {
        let bundle = Bundle(for: FixtureMarker.self)
        let url = try #require(bundle.url(forResource: name, withExtension: "json"))
        return try Data(contentsOf: url)
    }
}


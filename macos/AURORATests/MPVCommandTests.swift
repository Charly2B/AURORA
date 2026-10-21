import Foundation
import Testing
@testable import AURORA

@Suite("Commandes libmpv")
struct MPVCommandTests {
    @Test func loadMapsHeadersAndRedactsSecretsFromDescription() throws {
        let request = MPVLoadRequest(
            url: try #require(URL(string: "http://nas:8890/video.mkv")),
            headers: ["Cookie": "aurora_session=top-secret", "X-Aurora": "1"],
            start: 12.5
        )

        #expect(request.headerFieldOption.contains("Cookie: aurora_session=top-secret"))
        #expect(request.headerFieldOption.contains("X-Aurora: 1"))
        #expect(!request.description.contains("top-secret"))
        #expect(request.commands == [
            MPVCommand.setProperty("start", "12.5"),
            MPVCommand.loadFile("http://nas:8890/video.mkv")
        ])
    }

    @Test func selectionUsesMpvTrackIDsNotArrayOffsets() {
        #expect(MPVCommand.selectAudio(id: 17).arguments == ["set", "aid", "17"])
        #expect(MPVCommand.selectSubtitle(id: 43).arguments == ["set", "sid", "43"])
        #expect(MPVCommand.selectSubtitle(id: nil).arguments == ["set", "sid", "no"])
    }
}

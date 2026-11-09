import Testing
@testable import AURORA

@Suite("Présentation de l’accueil")
struct HomePresentationTests {
    @Test func displaysOnlyTheApprovedTitle() {
        #expect(HomePresentation.title == "MEDIATHEQUE AURORA")
        #expect(HomePresentation.showsMediaThumbnails == false)
    }
}

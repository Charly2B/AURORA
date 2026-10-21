import Foundation
import Testing
@testable import AURORA

@Suite("Mémoire navigation")
@MainActor
struct BrowseMemoryTests {
    @Test func restoresStateByServerTabAndFolderThenPurgesOnLeave() {
        let memory = BrowseMemory()
        let key = BrowseKey(origin: "http://nas:8890", tab: .movies, libraryID: 12)
        let state = BrowseState(sort: .rating, grouped: true, selectedLibraryID: 12, scrollTarget: 42, openedItemID: 99)

        memory.save(state, for: key)
        #expect(memory.restore(key)?.sort == .rating)
        #expect(memory.restore(key)?.scrollTarget == 42)
        #expect(memory.restore(key)?.grouped == true)
        #expect(memory.restore(key)?.selectedLibraryID == 12)

        memory.leave(key)
        #expect(memory.restore(key) == nil)
    }
}

func decode<T: Decodable>(_ json: String) -> T {
    try! JSONDecoder().decode(T.self, from: Data(json.utf8))
}

func card(_ id: Int64, _ title: String) -> Card {
    decode(#"{"id":\#(id),"kind":"movie","title":"\#(title)"}"#)
}

func page(_ cards: [Card], total: Int) -> ListPage {
    let data = try! JSONEncoder().encode(cards)
    let items = try! JSONSerialization.jsonObject(with: data)
    let page = try! JSONSerialization.data(withJSONObject: ["items": items, "total": total])
    return try! JSONDecoder().decode(ListPage.self, from: page)
}

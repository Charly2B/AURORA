import Foundation

enum LibraryOrderStore {
    private static let prefix = "aurora.libraryOrder."

    static func ordered(_ libraries: [LibraryRef], origin: String, kind: MediaKind) -> [LibraryRef] {
        let saved = load(origin: origin, kind: kind)
        let byID = Dictionary(uniqueKeysWithValues: libraries.map { ($0.id, $0) })
        let known = saved.compactMap { byID[$0] }
        let knownIDs = Set(known.map(\.id))
        return known + libraries.filter { !knownIDs.contains($0.id) }
    }

    static func move(
        sourceID: Int64,
        before destinationID: Int64,
        libraries: [LibraryRef],
        origin: String,
        kind: MediaKind
    ) {
        var ids = ordered(libraries, origin: origin, kind: kind).map(\.id)
        guard let sourceIndex = ids.firstIndex(of: sourceID), let destinationIndex = ids.firstIndex(of: destinationID), sourceID != destinationID else { return }
        let moved = ids.remove(at: sourceIndex)
        let insertionIndex = min(destinationIndex, ids.count)
        ids.insert(moved, at: insertionIndex)
        save(ids, origin: origin, kind: kind)
    }

    private static func load(origin: String, kind: MediaKind) -> [Int64] {
        UserDefaults.standard.array(forKey: key(origin: origin, kind: kind)) as? [Int64] ?? []
    }

    private static func save(_ ids: [Int64], origin: String, kind: MediaKind) {
        UserDefaults.standard.set(ids, forKey: key(origin: origin, kind: kind))
    }

    private static func key(origin: String, kind: MediaKind) -> String {
        let safeOrigin = Data(origin.utf8).base64EncodedString()
        return "\(prefix)\(safeOrigin).\(kind.rawValue)"
    }
}

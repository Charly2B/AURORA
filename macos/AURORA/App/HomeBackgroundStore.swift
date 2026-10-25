import AppKit
import Foundation

enum HomeBackgroundStore {
    static let pathKey = "aurora.homeBackgroundPath"

    static var directory: URL {
        FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("AURORA", isDirectory: true)
            .appendingPathComponent("Appearance", isDirectory: true)
    }

    static func copyImage(from source: URL) throws -> URL {
        let fileManager = FileManager.default
        try fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
        let ext = source.pathExtension.isEmpty ? "jpg" : source.pathExtension.lowercased()
        let destination = directory.appendingPathComponent("home-background.\(ext)")
        if fileManager.fileExists(atPath: destination.path) {
            try fileManager.removeItem(at: destination)
        }
        try fileManager.copyItem(at: source, to: destination)
        return destination
    }

    static func removeImage(at path: String) {
        guard !path.isEmpty else { return }
        try? FileManager.default.removeItem(atPath: path)
    }
}

import Foundation

actor ImageRepository {
    private let api: AuroraServing
    private let memory = NSCache<NSString, NSData>()
    private let bytes: URLCache

    init(
        api: AuroraServing,
        byteCache: URLCache = URLCache(
            memoryCapacity: 32 * 1024 * 1024,
            diskCapacity: 512 * 1024 * 1024,
            diskPath: "aurora-images"
        )
    ) {
        self.api = api
        bytes = byteCache
        memory.totalCostLimit = 128 * 1024 * 1024
    }

    func data(for path: String) async throws -> Data {
        let cacheURL = api.absoluteURL(path) ?? URL(string: "aurora-cache://image/\(path.hashValue)")!
        let request = URLRequest(url: cacheURL)
        let key = cacheURL.absoluteString as NSString
        if let cached = memory.object(forKey: key) { return cached as Data }
        if let cached = bytes.cachedResponse(for: request)?.data {
            memory.setObject(cached as NSData, forKey: key, cost: cached.count)
            return cached
        }
        let data = try await api.imageData(path: path)
        memory.setObject(data as NSData, forKey: key, cost: data.count)
        let response = URLResponse(url: cacheURL, mimeType: nil, expectedContentLength: data.count, textEncodingName: nil)
        bytes.storeCachedResponse(CachedURLResponse(response: response, data: data), for: request)
        return data
    }
}


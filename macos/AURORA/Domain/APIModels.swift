import Foundation

@propertyWrapper
struct DefaultFalse: Codable, Sendable {
    var wrappedValue = false
    init() {}
    init(wrappedValue: Bool) { self.wrappedValue = wrappedValue }
    init(from decoder: Decoder) throws { wrappedValue = try decoder.singleValueContainer().decode(Bool.self) }
    func encode(to encoder: Encoder) throws { var c = encoder.singleValueContainer(); try c.encode(wrappedValue) }
}

@propertyWrapper
struct DefaultTrue: Codable, Sendable {
    var wrappedValue = true
    init() {}
    init(wrappedValue: Bool) { self.wrappedValue = wrappedValue }
    init(from decoder: Decoder) throws { wrappedValue = try decoder.singleValueContainer().decode(Bool.self) }
    func encode(to encoder: Encoder) throws { var c = encoder.singleValueContainer(); try c.encode(wrappedValue) }
}

@propertyWrapper
struct DefaultString: Codable, Sendable {
    var wrappedValue = ""
    init() {}
    init(wrappedValue: String) { self.wrappedValue = wrappedValue }
    init(from decoder: Decoder) throws { wrappedValue = try decoder.singleValueContainer().decode(String.self) }
    func encode(to encoder: Encoder) throws { var c = encoder.singleValueContainer(); try c.encode(wrappedValue) }
}

@propertyWrapper
struct DefaultInt: Codable, Sendable {
    var wrappedValue = 0
    init() {}
    init(wrappedValue: Int) { self.wrappedValue = wrappedValue }
    init(from decoder: Decoder) throws { wrappedValue = try decoder.singleValueContainer().decode(Int.self) }
    func encode(to encoder: Encoder) throws { var c = encoder.singleValueContainer(); try c.encode(wrappedValue) }
}

@propertyWrapper
struct DefaultMinusOneInt: Codable, Sendable {
    var wrappedValue = -1
    init() {}
    init(wrappedValue: Int) { self.wrappedValue = wrappedValue }
    init(from decoder: Decoder) throws { wrappedValue = try decoder.singleValueContainer().decode(Int.self) }
    func encode(to encoder: Encoder) throws { var c = encoder.singleValueContainer(); try c.encode(wrappedValue) }
}

@propertyWrapper
struct DefaultDouble: Codable, Sendable {
    var wrappedValue = 0.0
    init() {}
    init(wrappedValue: Double) { self.wrappedValue = wrappedValue }
    init(from decoder: Decoder) throws { wrappedValue = try decoder.singleValueContainer().decode(Double.self) }
    func encode(to encoder: Encoder) throws { var c = encoder.singleValueContainer(); try c.encode(wrappedValue) }
}

@propertyWrapper
struct DefaultArray<Element: Codable & Sendable>: Codable, Sendable {
    var wrappedValue: [Element] = []
    init() {}
    init(wrappedValue: [Element]) { self.wrappedValue = wrappedValue }
    init(from decoder: Decoder) throws { wrappedValue = try decoder.singleValueContainer().decode([Element].self) }
    func encode(to encoder: Encoder) throws { var c = encoder.singleValueContainer(); try c.encode(wrappedValue) }
}

extension KeyedDecodingContainer {
    func decode(_ type: DefaultFalse.Type, forKey key: Key) throws -> DefaultFalse { try decodeIfPresent(type, forKey: key) ?? .init() }
    func decode(_ type: DefaultTrue.Type, forKey key: Key) throws -> DefaultTrue { try decodeIfPresent(type, forKey: key) ?? .init() }
    func decode(_ type: DefaultString.Type, forKey key: Key) throws -> DefaultString { try decodeIfPresent(type, forKey: key) ?? .init() }
    func decode(_ type: DefaultInt.Type, forKey key: Key) throws -> DefaultInt { try decodeIfPresent(type, forKey: key) ?? .init() }
    func decode(_ type: DefaultMinusOneInt.Type, forKey key: Key) throws -> DefaultMinusOneInt { try decodeIfPresent(type, forKey: key) ?? .init() }
    func decode(_ type: DefaultDouble.Type, forKey key: Key) throws -> DefaultDouble { try decodeIfPresent(type, forKey: key) ?? .init() }
    func decode<Element>(_ type: DefaultArray<Element>.Type, forKey key: Key) throws -> DefaultArray<Element> { try decodeIfPresent(type, forKey: key) ?? .init() }
}

struct Card: Codable, Sendable, Identifiable {
    let id: Int64
    let kind: String
    let title: String
    var year: Int?
    var poster: String?
    var backdrop: String?
    var thumb: String?
    var rating: Double?
    var runtime: Int?
    var duration: Double?
    var resolution: String?
    var hdr: String?
    @DefaultFalse var watched = false
    var position: Double?
    @DefaultFalse var favorite = false
    @DefaultFalse var watchlist = false
    var overview: String?
    var genres: [String]?
    @DefaultDouble var addedAt = 0
    var seriesId: Int64?
    var seriesTitle: String?
    var season: Int?
    var episode: Int?
    var episodeEnd: Int?
    var unwatched: Int?
    var total: Int?
    var versions: Int?

    var length: Double { duration ?? Double(runtime ?? 0) * 60 }
    var progress: Double { length > 0 ? min(max((position ?? 0) / length, 0), 1) : 0 }
}

struct Section: Codable, Sendable, Identifiable {
    let id: String
    let title: String
    var libraryId: Int64?
    var kind: String?
    @DefaultArray<Card> var items = []
}

struct Home: Codable, Sendable {
    var hero: Card?
    @DefaultArray<Section> var sections = []
}

struct LibraryRef: Codable, Sendable, Identifiable {
    let id: Int64
    let name: String
    @DefaultInt var movies = 0
    @DefaultInt var series = 0
}

struct ListPage: Codable, Sendable {
    @DefaultArray<Card> var items = []
    @DefaultInt var total = 0
}

struct AudioTrack: Codable, Sendable, Identifiable {
    let index: Int
    @DefaultMinusOneInt var stream = -1
    @DefaultString var codec = ""
    @DefaultString var profile = ""
    @DefaultInt var channels = 0
    @DefaultString var layout = ""
    @DefaultString var language = ""
    @DefaultString var title = ""
    @DefaultFalse var isDefault = false
    @DefaultFalse var atmos = false

    var id: Int { index }
    enum CodingKeys: String, CodingKey { case index, stream, codec, profile, channels, layout, language, title, isDefault = "default", atmos }
}

struct SubtitleTrack: Codable, Sendable, Identifiable {
    let index: Int
    @DefaultString var codec = ""
    @DefaultString var language = ""
    @DefaultString var title = ""
    @DefaultFalse var isDefault = false
    @DefaultFalse var forced = false
    @DefaultTrue var text = true

    var id: Int { index }
    enum CodingKeys: String, CodingKey { case index, codec, language, title, isDefault = "default", forced, text }
}

struct Chapter: Codable, Sendable, Identifiable {
    @DefaultDouble var start = 0
    @DefaultDouble var end = 0
    @DefaultString var title = ""
    var id: Double { start }
}

struct MediaFile: Codable, Sendable, Identifiable {
    let id: Int64
    @DefaultString var name = ""
    @DefaultString var label = ""
    var edition: String?
    var size: Int64?
    @DefaultString var container = ""
    @DefaultDouble var duration = 0
    var bitrate: Int?
    @DefaultString var videoCodec = ""
    @DefaultString var videoProfile = ""
    @DefaultInt var width = 0
    @DefaultInt var height = 0
    @DefaultString var resolution = ""
    @DefaultString var hdr = ""
    @DefaultArray<AudioTrack> var audio = []
    @DefaultArray<SubtitleTrack> var subtitles = []
    @DefaultArray<Chapter> var chapters = []
    var probeState: String?
    var probeError: String?
    var addedAt: Double?
}

struct Person: Codable, Sendable, Identifiable {
    let name: String
    let role: String
    var character: String?
    var photo: String?
    var id: String { "\(role):\(name):\(character ?? "")" }
}

struct ItemState: Codable, Sendable {
    @DefaultDouble var position = 0
    @DefaultDouble var duration = 0
    @DefaultFalse var watched = false
    @DefaultInt var playCount = 0
    var lastPlayedAt: Double?
    var lastFileId: Int64?
    @DefaultFalse var favorite = false
    @DefaultFalse var watchlist = false
}

struct Season: Codable, Sendable, Identifiable {
    let number: Int
    let title: String
    var overview: String?
    var poster: String?
    @DefaultInt var episodes = 0
    @DefaultInt var unwatched = 0
    var id: Int { number }
}

struct Detail: Codable, Sendable, Identifiable {
    let id: Int64
    let kind: String
    let title: String
    var year: Int?
    var poster: String?
    var backdrop: String?
    var thumb: String?
    var rating: Double?
    var ratingSource: String?
    var runtime: Int?
    var duration: Double?
    var resolution: String?
    var hdr: String?
    var overview: String?
    var tagline: String?
    var originalTitle: String?
    var contentRating: String?
    var status: String?
    var airDate: String?
    var trailer: String?
    @DefaultString var matchState = ""
    var tmdbId: Int64?
    @DefaultInt var libraryId = 0
    var genres: [String]?
    var seriesId: Int64?
    var seriesTitle: String?
    var season: Int?
    var episode: Int?
    var episodeEnd: Int?
    var total: Int?
    var unwatched: Int?
    @DefaultFalse var watched = false
    @DefaultFalse var favorite = false
    @DefaultFalse var watchlist = false
    @DefaultArray<MediaFile> var files = []
    @DefaultArray<Person> var people = []
    var state: ItemState = .init()
    var seasons: [Season]?
    var nextUp: Card?
    var collections: [Card]?
    var similar: [Card]?

    enum CodingKeys: String, CodingKey {
        case id, kind, title, year, poster, backdrop, thumb, rating, ratingSource, runtime, duration, resolution, hdr
        case overview, tagline, originalTitle, contentRating, status, airDate, trailer, matchState, tmdbId, libraryId, genres
        case seriesId, seriesTitle, season, episode, episodeEnd, total, unwatched, watched, favorite, watchlist, files, people
        case state, seasons, nextUp, collections, similar
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(Int64.self, forKey: .id)
        kind = try c.decode(String.self, forKey: .kind)
        title = try c.decode(String.self, forKey: .title)
        year = try c.decodeIfPresent(Int.self, forKey: .year)
        poster = try c.decodeIfPresent(String.self, forKey: .poster)
        backdrop = try c.decodeIfPresent(String.self, forKey: .backdrop)
        thumb = try c.decodeIfPresent(String.self, forKey: .thumb)
        rating = try c.decodeIfPresent(Double.self, forKey: .rating)
        ratingSource = try c.decodeIfPresent(String.self, forKey: .ratingSource)
        runtime = try c.decodeIfPresent(Int.self, forKey: .runtime)
        duration = try c.decodeIfPresent(Double.self, forKey: .duration)
        resolution = try c.decodeIfPresent(String.self, forKey: .resolution)
        hdr = try c.decodeIfPresent(String.self, forKey: .hdr)
        overview = try c.decodeIfPresent(String.self, forKey: .overview)
        tagline = try c.decodeIfPresent(String.self, forKey: .tagline)
        originalTitle = try c.decodeIfPresent(String.self, forKey: .originalTitle)
        contentRating = try c.decodeIfPresent(String.self, forKey: .contentRating)
        status = try c.decodeIfPresent(String.self, forKey: .status)
        airDate = try c.decodeIfPresent(String.self, forKey: .airDate)
        trailer = try c.decodeIfPresent(String.self, forKey: .trailer)
        _matchState = try c.decode(DefaultString.self, forKey: .matchState)
        tmdbId = try c.decodeIfPresent(Int64.self, forKey: .tmdbId)
        _libraryId = try c.decode(DefaultInt.self, forKey: .libraryId)
        genres = try c.decodeIfPresent([String].self, forKey: .genres)
        seriesId = try c.decodeIfPresent(Int64.self, forKey: .seriesId)
        seriesTitle = try c.decodeIfPresent(String.self, forKey: .seriesTitle)
        season = try c.decodeIfPresent(Int.self, forKey: .season)
        episode = try c.decodeIfPresent(Int.self, forKey: .episode)
        episodeEnd = try c.decodeIfPresent(Int.self, forKey: .episodeEnd)
        total = try c.decodeIfPresent(Int.self, forKey: .total)
        unwatched = try c.decodeIfPresent(Int.self, forKey: .unwatched)
        _watched = try c.decode(DefaultFalse.self, forKey: .watched)
        _favorite = try c.decode(DefaultFalse.self, forKey: .favorite)
        _watchlist = try c.decode(DefaultFalse.self, forKey: .watchlist)
        _files = try c.decode(DefaultArray<MediaFile>.self, forKey: .files)
        _people = try c.decode(DefaultArray<Person>.self, forKey: .people)
        state = try c.decodeIfPresent(ItemState.self, forKey: .state) ?? .init()
        seasons = try c.decodeIfPresent([Season].self, forKey: .seasons)
        nextUp = try c.decodeIfPresent(Card.self, forKey: .nextUp)
        collections = try c.decodeIfPresent([Card].self, forKey: .collections)
        similar = try c.decodeIfPresent([Card].self, forKey: .similar)
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(id, forKey: .id); try c.encode(kind, forKey: .kind); try c.encode(title, forKey: .title)
        try c.encodeIfPresent(year, forKey: .year); try c.encodeIfPresent(poster, forKey: .poster); try c.encodeIfPresent(backdrop, forKey: .backdrop)
        try c.encodeIfPresent(thumb, forKey: .thumb); try c.encodeIfPresent(rating, forKey: .rating); try c.encodeIfPresent(ratingSource, forKey: .ratingSource)
        try c.encodeIfPresent(runtime, forKey: .runtime); try c.encodeIfPresent(duration, forKey: .duration); try c.encodeIfPresent(resolution, forKey: .resolution)
        try c.encodeIfPresent(hdr, forKey: .hdr); try c.encodeIfPresent(overview, forKey: .overview); try c.encodeIfPresent(tagline, forKey: .tagline)
        try c.encodeIfPresent(originalTitle, forKey: .originalTitle); try c.encodeIfPresent(contentRating, forKey: .contentRating); try c.encodeIfPresent(status, forKey: .status)
        try c.encodeIfPresent(airDate, forKey: .airDate); try c.encodeIfPresent(trailer, forKey: .trailer); try c.encode(matchState, forKey: .matchState)
        try c.encodeIfPresent(tmdbId, forKey: .tmdbId); try c.encode(libraryId, forKey: .libraryId); try c.encodeIfPresent(genres, forKey: .genres)
        try c.encodeIfPresent(seriesId, forKey: .seriesId); try c.encodeIfPresent(seriesTitle, forKey: .seriesTitle); try c.encodeIfPresent(season, forKey: .season)
        try c.encodeIfPresent(episode, forKey: .episode); try c.encodeIfPresent(episodeEnd, forKey: .episodeEnd); try c.encodeIfPresent(total, forKey: .total)
        try c.encodeIfPresent(unwatched, forKey: .unwatched); try c.encode(watched, forKey: .watched); try c.encode(favorite, forKey: .favorite)
        try c.encode(watchlist, forKey: .watchlist); try c.encode(files, forKey: .files); try c.encode(people, forKey: .people); try c.encode(state, forKey: .state)
        try c.encodeIfPresent(seasons, forKey: .seasons); try c.encodeIfPresent(nextUp, forKey: .nextUp); try c.encodeIfPresent(collections, forKey: .collections)
        try c.encodeIfPresent(similar, forKey: .similar)
    }
}

struct PlaySubtitle: Codable, Sendable, Identifiable {
    let index: Int
    @DefaultString var label = ""
    @DefaultString var language = ""
    @DefaultString var codec = ""
    @DefaultFalse var forced = false
    @DefaultFalse var isDefault = false
    @DefaultTrue var text = true
    @DefaultFalse var external = false
    var url: String?
    var id: Int { index }
    enum CodingKeys: String, CodingKey { case index, label, language, codec, forced, isDefault = "default", text, external, url }
}

struct PlayResponse: Codable, Sendable {
    let mode: String
    @DefaultString var reason = ""
    let url: String
    var sessionId: String?
    @DefaultDouble var offset = 0
    @DefaultDouble var start = 0
    @DefaultDouble var duration = 0
    @DefaultArray<Chapter> var chapters = []
    @DefaultDouble var itemIdValue = 0
    @DefaultDouble var fileIdValue = 0
    @DefaultArray<AudioTrack> var audio = []
    @DefaultMinusOneInt var audioStream = -1
    @DefaultArray<PlaySubtitle> var subtitles = []
    @DefaultFalse var burnedSubtitle = false
    var signedUrl: String?

    var itemId: Int64 { Int64(itemIdValue) }
    var fileId: Int64 { Int64(fileIdValue) }
    enum CodingKeys: String, CodingKey {
        case mode, reason, url, sessionId, offset, start, duration, chapters, itemIdValue = "itemId", fileIdValue = "fileId"
        case audio, audioStream, subtitles, burnedSubtitle, signedUrl
    }
}

struct SearchResults: Codable, Sendable {
    @DefaultArray<Card> var movies = []
    @DefaultArray<Card> var series = []
    @DefaultArray<Card> var episodes = []
    @DefaultArray<String> var people = []
    @DefaultFalse var fuzzy = false
}

struct Preferences: Codable, Sendable {
    @DefaultTrue var autoplayNext = true
    var audioLang = "fre"
    @DefaultString var subLang = ""
    var subMode = "auto"
    var subSize = "m"

    enum CodingKeys: String, CodingKey { case autoplayNext, audioLang, subLang, subMode, subSize }
    init() {}
    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        _autoplayNext = try c.decode(DefaultTrue.self, forKey: .autoplayNext)
        audioLang = try c.decodeIfPresent(String.self, forKey: .audioLang) ?? "fre"
        _subLang = try c.decode(DefaultString.self, forKey: .subLang)
        subMode = try c.decodeIfPresent(String.self, forKey: .subMode) ?? "auto"
        subSize = try c.decodeIfPresent(String.self, forKey: .subSize) ?? "m"
    }
}

struct UserProfile: Codable, Sendable, Identifiable {
    let id: Int64
    let username: String
    let role: String
    var maxAge: Int?
    @DefaultFalse var hideUnrated = false
    var prefs: Preferences = .init()

    enum CodingKeys: String, CodingKey { case id, username, role, maxAge, hideUnrated, prefs }
    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(Int64.self, forKey: .id)
        username = try c.decode(String.self, forKey: .username)
        role = try c.decode(String.self, forKey: .role)
        maxAge = try c.decodeIfPresent(Int.self, forKey: .maxAge)
        _hideUnrated = try c.decode(DefaultFalse.self, forKey: .hideUnrated)
        prefs = try c.decodeIfPresent(Preferences.self, forKey: .prefs) ?? .init()
    }
}

struct Status: Codable, Sendable {
    @DefaultString var version = ""
    @DefaultFalse var hasAdmin = false
    @DefaultFalse var setupDone = false
    var user: UserProfile?
}

struct Capabilities: Codable, Sendable {
    let containers: [String]
    let video: [String]
    let audio: [String]
    let hls: String
    let maxHeight: Int
}

struct LinkStart: Codable, Sendable {
    let code: String
    let poll: String
    var expiresIn = 600

    enum CodingKeys: String, CodingKey { case code, poll, expiresIn }
    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        code = try c.decode(String.self, forKey: .code)
        poll = try c.decode(String.self, forKey: .poll)
        expiresIn = try c.decodeIfPresent(Int.self, forKey: .expiresIn) ?? 600
    }
}

struct PlayRequest: Codable, Sendable {
    let fileId: Int64
    let caps: Capabilities
    var start = 0.0
    var audioIndex: Int?
    var subtitleIndex: Int?
    var forceMode: String?
    var external = false
}


import CoreMedia
import Testing
@testable import AURORA

@Suite("Capacités libmpv")
struct PlaybackCapabilitiesTests {
    @Test func h264AlwaysAvailableAndHardwareCodecsAreGated() {
        let none = PlaybackCapabilities.detect { _ in false }
        #expect(none.containers == ["mp4", "mov", "mkv", "webm", "ts", "avi"])
        #expect(none.video.contains("h264"))
        #expect(!none.video.contains("hevc"))
        #expect(!none.video.contains("av1"))
        #expect(none.hls == "native")
        #expect(none.maxHeight <= 2160)

        let supported = PlaybackCapabilities.detect { codec in
            codec == kCMVideoCodecType_HEVC || codec == kCMVideoCodecType_AV1
        }
        #expect(supported.video.contains("hevc"))
        #expect(supported.video.contains("av1"))
        #expect(Set(["aac", "ac3", "eac3", "dts", "flac", "opus"]).isSubset(of: Set(supported.audio)))
    }
}

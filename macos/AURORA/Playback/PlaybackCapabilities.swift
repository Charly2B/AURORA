import CoreMedia
import VideoToolbox

enum PlaybackCapabilities {
    static func detect(
        hardwareDecoder: (CMVideoCodecType) -> Bool = { VTIsHardwareDecodeSupported($0) }
    ) -> Capabilities {
        var video = ["h264"]
        if hardwareDecoder(kCMVideoCodecType_HEVC) { video.append("hevc") }
        if hardwareDecoder(kCMVideoCodecType_AV1) { video.append("av1") }
        return Capabilities(
            containers: ["mp4", "mov", "mkv", "webm", "ts", "avi"],
            video: video,
            audio: ["aac", "ac3", "eac3", "dts", "flac", "opus", "mp3", "vorbis", "alac", "pcm"],
            hls: "native",
            maxHeight: 2160
        )
    }
}

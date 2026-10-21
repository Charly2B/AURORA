import AppKit
import AVFoundation
import Testing
@testable import AURORA

@Suite("Smoke libmpv", .serialized)
@MainActor
struct LibMPVSmokeTests {
    @Test func playsLocalH264AACToEndWithoutNetwork() async throws {
        let fixture = FileManager.default.temporaryDirectory
            .appendingPathComponent("aurora-mpv-\(UUID().uuidString).mp4")
        defer { try? FileManager.default.removeItem(at: fixture) }
        try await makeFixture(at: fixture)

        let window = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 640, height: 360), styleMask: [.borderless], backing: .buffered, defer: false)
        let view = NSView(frame: window.contentView!.bounds)
        window.contentView = view
        let controller = LibMPVController()
        controller.attach(to: view)

        var reachedReady = false
        var reachedEnd = false
        let timeout = Task { @MainActor in
            try? await Task.sleep(for: .seconds(8))
            if !Task.isCancelled { controller.stop() }
        }
        controller.load(url: fixture, headers: [:], start: 0)
        for await event in controller.events {
            if event == .ready { reachedReady = true }
            if event == .ended { reachedEnd = true; break }
            if case .failed(let failure) = event {
                Issue.record("libmpv: \(failure.message)")
                break
            }
        }
        timeout.cancel()
        controller.stop()

        #expect(reachedReady)
        #expect(reachedEnd)
    }

    private func makeFixture(at url: URL) async throws {
        let writer = try AVAssetWriter(outputURL: url, fileType: .mp4)
        let video = AVAssetWriterInput(mediaType: .video, outputSettings: [
            AVVideoCodecKey: AVVideoCodecType.h264,
            AVVideoWidthKey: 320,
            AVVideoHeightKey: 180
        ])
        let adaptor = AVAssetWriterInputPixelBufferAdaptor(
            assetWriterInput: video,
            sourcePixelBufferAttributes: [
                kCVPixelBufferPixelFormatTypeKey as String: kCVPixelFormatType_32BGRA,
                kCVPixelBufferWidthKey as String: 320,
                kCVPixelBufferHeightKey as String: 180
            ]
        )
        let audio = AVAssetWriterInput(mediaType: .audio, outputSettings: [
            AVFormatIDKey: kAudioFormatMPEG4AAC,
            AVSampleRateKey: 44_100,
            AVNumberOfChannelsKey: 1,
            AVEncoderBitRateKey: 96_000
        ])
        #expect(writer.canAdd(video))
        #expect(writer.canAdd(audio))
        writer.add(video)
        writer.add(audio)
        #expect(writer.startWriting())
        writer.startSession(atSourceTime: .zero)

        for frame in 0..<60 {
            while !video.isReadyForMoreMediaData { try await Task.sleep(for: .milliseconds(2)) }
            var buffer: CVPixelBuffer?
            guard let pool = adaptor.pixelBufferPool,
                  CVPixelBufferPoolCreatePixelBuffer(nil, pool, &buffer) == kCVReturnSuccess,
                  let buffer else { throw FixtureError.pixelBuffer }
            CVPixelBufferLockBaseAddress(buffer, [])
            if let base = CVPixelBufferGetBaseAddress(buffer) {
                memset(base, frame % 2 == 0 ? 0x18 : 0x28, CVPixelBufferGetDataSize(buffer))
            }
            CVPixelBufferUnlockBaseAddress(buffer, [])
            #expect(adaptor.append(buffer, withPresentationTime: CMTime(value: CMTimeValue(frame), timescale: 30)))
        }
        video.markAsFinished()

        let audioBuffer = try silentAudioBuffer(seconds: 2)
        while !audio.isReadyForMoreMediaData { try await Task.sleep(for: .milliseconds(2)) }
        #expect(audio.append(audioBuffer))
        audio.markAsFinished()
        await writer.finishWriting()
        if let error = writer.error { throw error }
        #expect(writer.status == .completed)
    }

    private func silentAudioBuffer(seconds: Int) throws -> CMSampleBuffer {
        let sampleRate: Int32 = 44_100
        let frames = Int(sampleRate) * seconds
        let byteCount = frames * MemoryLayout<Int16>.size
        var block: CMBlockBuffer?
        try check(CMBlockBufferCreateWithMemoryBlock(
            allocator: kCFAllocatorDefault,
            memoryBlock: nil,
            blockLength: byteCount,
            blockAllocator: kCFAllocatorDefault,
            customBlockSource: nil,
            offsetToData: 0,
            dataLength: byteCount,
            flags: 0,
            blockBufferOut: &block
        ))
        let silence = Data(count: byteCount)
        try silence.withUnsafeBytes { bytes in
            try check(CMBlockBufferReplaceDataBytes(with: bytes.baseAddress!, blockBuffer: block!, offsetIntoDestination: 0, dataLength: byteCount))
        }

        var asbd = AudioStreamBasicDescription(
            mSampleRate: Double(sampleRate),
            mFormatID: kAudioFormatLinearPCM,
            mFormatFlags: kLinearPCMFormatFlagIsSignedInteger | kLinearPCMFormatFlagIsPacked,
            mBytesPerPacket: 2,
            mFramesPerPacket: 1,
            mBytesPerFrame: 2,
            mChannelsPerFrame: 1,
            mBitsPerChannel: 16,
            mReserved: 0
        )
        var format: CMAudioFormatDescription?
        try check(CMAudioFormatDescriptionCreate(
            allocator: kCFAllocatorDefault,
            asbd: &asbd,
            layoutSize: 0,
            layout: nil,
            magicCookieSize: 0,
            magicCookie: nil,
            extensions: nil,
            formatDescriptionOut: &format
        ))
        var timing = CMSampleTimingInfo(
            duration: CMTime(value: 1, timescale: sampleRate),
            presentationTimeStamp: .zero,
            decodeTimeStamp: .invalid
        )
        var sample: CMSampleBuffer?
        try check(CMSampleBufferCreate(
            allocator: kCFAllocatorDefault,
            dataBuffer: block,
            dataReady: true,
            makeDataReadyCallback: nil,
            refcon: nil,
            formatDescription: format,
            sampleCount: frames,
            sampleTimingEntryCount: 1,
            sampleTimingArray: &timing,
            sampleSizeEntryCount: 0,
            sampleSizeArray: nil,
            sampleBufferOut: &sample
        ))
        return sample!
    }

    private func check(_ status: OSStatus) throws {
        if status != noErr { throw FixtureError.coreMedia(status) }
    }

    private enum FixtureError: Error {
        case pixelBuffer
        case coreMedia(OSStatus)
    }
}

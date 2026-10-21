import AppKit
import Testing
@testable import AURORA

@Suite("Clavier lecteur")
@MainActor
struct PlayerKeyCommandTests {
    @Test(arguments: [
        (49, " ", PlayerKeyCommand.togglePause),
        (123, "", PlayerKeyCommand.seek(-10)),
        (124, "", PlayerKeyCommand.seek(30)),
        (3, "f", PlayerKeyCommand.fullscreen),
        (53, "\u{1b}", PlayerKeyCommand.closePanelOrPlayer),
        (8, "c", PlayerKeyCommand.cycleSubtitles),
        (0, "a", PlayerKeyCommand.cycleAudio)
    ])
    func mapsPlaybackKeys(keyCode: Int, characters: String, expected: PlayerKeyCommand) throws {
        let event = try #require(NSEvent.keyEvent(
            with: .keyDown,
            location: .zero,
            modifierFlags: [],
            timestamp: 0,
            windowNumber: 0,
            context: nil,
            characters: characters,
            charactersIgnoringModifiers: characters,
            isARepeat: false,
            keyCode: UInt16(keyCode)
        ))
        #expect(PlayerKeyCommand.map(event, firstResponder: nil) == expected)
    }

    @Test func textInputIgnoresPlaybackKeys() throws {
        let field = NSTextField()
        let event = try #require(NSEvent.keyEvent(
            with: .keyDown, location: .zero, modifierFlags: [], timestamp: 0,
            windowNumber: 0, context: nil, characters: " ", charactersIgnoringModifiers: " ",
            isARepeat: false, keyCode: 49
        ))
        #expect(PlayerKeyCommand.map(event, firstResponder: field) == nil)
    }

    @Test func controlsHideOnlyWhilePlayingWithoutMenu() {
        #expect(PlayerControlsVisibility.shouldAutoHide(playback: .playing, menuOpen: false))
        #expect(!PlayerControlsVisibility.shouldAutoHide(playback: .paused, menuOpen: false))
        #expect(!PlayerControlsVisibility.shouldAutoHide(playback: .playing, menuOpen: true))
    }
}

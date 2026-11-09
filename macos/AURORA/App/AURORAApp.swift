import SwiftUI

@main
struct AURORAApp: App {
    @NSApplicationDelegateAdaptor(AURORAAppDelegate.self) private var appDelegate

    var body: some Scene {
        WindowGroup {
            RootView()
        }
        .defaultSize(width: 1280, height: 800)
        .windowResizability(.contentMinSize)
        Settings {
            AppearanceSettingsView()
        }
        .commands {
            CommandGroup(after: .appInfo) {
                Button("Plein écran") { NSApp.keyWindow?.toggleFullScreen(nil) }
                    .keyboardShortcut("f", modifiers: [.command, .control])
            }
            CommandGroup(after: .textEditing) {
                Button("Rechercher") {
                    NotificationCenter.default.post(name: .auroraSearchRequested, object: nil)
                }
                .keyboardShortcut("f", modifiers: .command)
            }
        }
    }
}

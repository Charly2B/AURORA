import AppKit
import SwiftUI

struct MPVRenderView: NSViewRepresentable {
    let controller: any MPVControlling

    func makeNSView(context: Context) -> NSView {
        let view = NSView()
        view.wantsLayer = true
        view.layer?.backgroundColor = NSColor.black.cgColor
        controller.attach(to: view)
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {}
}

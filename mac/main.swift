// macOS shell for the router: LaunchServices hands links to an app as Apple Events, not as
// arguments, so this app receives them and passes each one to the bundled Go binary.
import Cocoa

final class Delegate: NSObject, NSApplicationDelegate {
    private var quitTimer: Timer?

    func application(_ application: NSApplication, open urls: [URL]) {
        let router = Bundle.main.bundleURL.appendingPathComponent("Contents/MacOS/router")
        for url in urls {
            let p = Process()
            p.executableURL = router
            p.arguments = [url.absoluteString]
            try? p.run()
            p.waitUntilExit()
        }
        scheduleQuit()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        let args = CommandLine.arguments
        if args.contains("--register") {
            setDefault(Bundle.main.bundleURL)
        } else if let i = args.firstIndex(of: "--restore"), i + 1 < args.count,
                  let app = NSWorkspace.shared.urlForApplication(withBundleIdentifier: args[i + 1]) {
            setDefault(app)
        } else {
            scheduleQuit()
        }
    }

    // macOS asks the user to confirm each change of the default browser; that dialog is intended.
    private func setDefault(_ app: URL) {
        let group = DispatchGroup()
        for scheme in ["http", "https"] {
            group.enter()
            NSWorkspace.shared.setDefaultApplication(at: app, toOpenURLsWithScheme: scheme) { error in
                if let error { NSLog("set default for \(scheme): \(error)") }
                group.leave()
            }
        }
        group.notify(queue: .main) { NSApp.terminate(nil) }
    }

    // Stay briefly so a burst of links is handled by one process.
    private func scheduleQuit() {
        quitTimer?.invalidate()
        quitTimer = Timer.scheduledTimer(withTimeInterval: 2, repeats: false) { _ in NSApp.terminate(nil) }
    }
}

let delegate = Delegate()
NSApplication.shared.delegate = delegate
NSApplication.shared.setActivationPolicy(.prohibited)
NSApplication.shared.run()

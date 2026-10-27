// macOS shell for the router: LaunchServices hands links to an app as Apple Events, not as
// arguments, so this app receives them and passes each one to the bundled Go binary.
import Cocoa

// Queried by `browser-router status`: which app opens web links. Runs without the app loop.
if CommandLine.arguments.contains("--print-default") {
    for scheme in ["http", "https"] {
        let app = NSWorkspace.shared.urlForApplication(toOpen: URL(string: "\(scheme)://example.com")!)
        print(scheme, app.flatMap { Bundle(url: $0)?.bundleIdentifier } ?? "none")
    }
    exit(0)
}

final class Delegate: NSObject, NSApplicationDelegate {
    private var quitTimer: Timer?

    func application(_ application: NSApplication, open urls: [URL]) {
        let router = Bundle.main.bundleURL.appendingPathComponent("Contents/MacOS/browser-router")
        var target: String?
        for url in urls {
            let p = Process()
            p.executableURL = router
            p.arguments = [url.absoluteString]
            p.environment = ProcessInfo.processInfo.environment.merging(["BROWSER_ROUTER_SHELL": "1"]) { $1 }
            let out = Pipe()
            p.standardOutput = out
            try? p.run()
            let data = out.fileHandleForReading.readDataToEndOfFile()
            p.waitUntilExit()
            // The router names the app that got the link: "activate: <app>", the last one wins
            for line in String(decoding: data, as: UTF8.self).split(separator: "\n") where line.hasPrefix("activate: ") {
                target = String(line.dropFirst("activate: ".count))
            }
        }
        if let target { bringToFront(target) }
        scheduleQuit()
    }

    // macOS 14 lets an app come to the front only when the active app hands over its activation. This
    // app is the one macOS activated for the link, so it passes that on to the browser that got it;
    // otherwise the browser opens the tab behind the other windows.
    private func bringToFront(_ name: String) {
        let app = NSWorkspace.shared.runningApplications.first {
            $0.localizedName == name || $0.bundleURL?.deletingPathExtension().lastPathComponent == name
        }
        guard let app else { return }
        if #available(macOS 14, *) {
            NSApp.yieldActivation(to: app)
            app.activate(from: NSRunningApplication.current, options: [])
        } else {
            app.activate()
        }
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
// accessory, not prohibited: no Dock icon, but the app can be active and so hand its activation on
NSApplication.shared.setActivationPolicy(.accessory)
NSApplication.shared.run()

import AppKit
import Foundation
import ServiceManagement

if CommandLine.arguments.contains("--version") {
    print("Noria " + (Bundle.main.object(forInfoDictionaryKey: "NoriaVersion") as? String ?? "unknown")); exit(0)
}
if CommandLine.arguments.contains("--register-login-item") || CommandLine.arguments.contains("--unregister-login-item") {
    do {
        if CommandLine.arguments.contains("--register-login-item") { try SMAppService.mainApp.register() }
        else { try SMAppService.mainApp.unregister() }
        exit(0)
    } catch { fputs("Login item could not be changed: \(error.localizedDescription)\n", stderr); exit(1) }
}
if CommandLine.arguments.contains("--login-item-status") { print(SMAppService.mainApp.status == .enabled ? "enabled" : "disabled"); exit(0) }
if CommandLine.arguments.contains("--check") {
    let file = cliURL()
    guard FileManager.default.isExecutableFile(atPath: file.path) else { fputs("Sibling noria CLI is missing\n", stderr); exit(1) }
    print("Native menu bar bundle and sibling CLI are ready"); exit(0)
}
if CommandLine.arguments.count == 3 && CommandLine.arguments[1] == "--validate-status-json" {
    do {
        let state = try decodeJSON(Dashboard.self, from: Data(contentsOf: URL(fileURLWithPath: CommandLine.arguments[2])))
        guard state.accounts.first?.title == "alex@example.com",
              let windows = state.accounts.first?.snapshot?.windows,
              windowText(windows.first, title: "5h").contains("75% left"),
              windowText(windows.last, title: "Week").contains("5% left") else { exit(1) }
        print("Native menu status decoder: emails, fractional timestamps, and quota percentages passed")
        exit(0)
    } catch { fputs("Invalid CLI status schema: \(error.localizedDescription)\n", stderr); exit(1) }
}
let home = selectedHome()
do { try FileManager.default.createDirectory(atPath: home, withIntermediateDirectories: true, attributes: [.posixPermissions: 0o700]) }
catch { fputs("Cannot open the Noria private home.\n", stderr); exit(1) }
guard let lock = SingleInstanceLock(home: home) else { exit(0) }
let app = NSApplication.shared
app.setActivationPolicy(.accessory)
let delegate = AppDelegate(cli: CLI(executable: cliURL(), home: home))
app.delegate = delegate
withExtendedLifetime(lock) { app.run() }

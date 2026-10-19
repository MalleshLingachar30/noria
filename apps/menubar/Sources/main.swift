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
if CommandLine.arguments.contains("--validate-status") {
    do { let data = FileHandle.standardInput.readDataToEndOfFile(); let status = try decodeJSON(Dashboard.self, from: data); print("Status schema \(status.schema_version), \(status.accounts.count) accounts, \(status.client_roots.count) client roots"); exit(0) }
    catch { fputs("Status schema invalid: \(error)\n", stderr); exit(1) }
}
if CommandLine.arguments.contains("--check") {
    let file = cliURL()
    guard FileManager.default.isExecutableFile(atPath: file.path) else { fputs("Sibling noria CLI is missing\n", stderr); exit(1) }
    print("Native menu bar bundle and sibling CLI are ready"); exit(0)
}
let home = selectedHome()
guard let lock = SingleInstanceLock(home: home) else { exit(0) }
let app = NSApplication.shared
app.setActivationPolicy(.accessory)
let delegate = AppDelegate(cli: CLI(executable: cliURL(), home: home))
app.delegate = delegate
withExtendedLifetime(lock) { app.run() }

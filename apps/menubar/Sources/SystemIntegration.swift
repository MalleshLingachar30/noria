import AppKit
import Darwin
import Foundation
import ServiceManagement

final class SingleInstanceLock {
    private let descriptor: Int32
    init?(home: String) {
        // One menu per user. The CLI supports multiple homes; launch an explicit
        // home when opening this UI rather than silently switching its pool.
        let path = "/tmp/noria-menubar-\(getuid()).lock"
        let descriptor = path.withCString { Darwin.open($0, O_CREAT | O_RDWR | O_NOFOLLOW, mode_t(S_IRUSR | S_IWUSR)) }
        guard descriptor >= 0 else { return nil }
        guard flock(descriptor, LOCK_EX | LOCK_NB) == 0 else { Darwin.close(descriptor); return nil }
        self.descriptor = descriptor
    }
    deinit { flock(descriptor, LOCK_UN); Darwin.close(descriptor) }
}
func cliURL() -> URL {
    let directory = Bundle.main.bundleURL.standardizedFileURL.resolvingSymlinksInPath().deletingLastPathComponent()
    for name in ["noria", "noria-bin"] {
        let candidate = directory.appendingPathComponent(name)
        if FileManager.default.isExecutableFile(atPath: candidate.path) { return candidate }
    }
    return directory.appendingPathComponent("noria")
}
func selectedHome() -> String {
    let args = CommandLine.arguments
    if let i = args.firstIndex(of: "--home"), i + 1 < args.count { return URL(fileURLWithPath: args[i + 1]).standardizedFileURL.path }
    if let value = ProcessInfo.processInfo.environment["NORIA_HOME"], !value.isEmpty { return value }
    return FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".noria").path
}
struct CommandResult {
    let status: Int32
    let output: Data
    let error: String
    var text: String { String(data: output, encoding: .utf8) ?? "" }
}
final class CLI {
    let executable: URL
    let home: String
    init(executable: URL, home: String) { self.executable = executable; self.home = home }
    // stdout and stderr are drained concurrently so large status/error output
    // cannot deadlock a background operation. No command runs on the UI thread.
    @discardableResult
    func run(_ arguments: [String], timeout: TimeInterval = 35, completion: @escaping (CommandResult) -> Void) -> Process {
        let process = Process()
        let output = Pipe(), errors = Pipe()
        process.executableURL = executable
        process.arguments = arguments + ["--home", home]
        process.standardOutput = output
        process.standardError = errors
        process.standardInput = FileHandle.nullDevice
        process.currentDirectoryURL = URL(fileURLWithPath: home)
        var env = ProcessInfo.processInfo.environment
        let user = FileManager.default.homeDirectoryForCurrentUser.path
        env["PATH"] = [user + "/.local/bin", user + "/.vite-plus/bin", "/opt/homebrew/bin", "/usr/local/bin", env["PATH"] ?? "/usr/bin:/bin:/usr/sbin:/sbin"].joined(separator: ":")
        process.environment = env
        DispatchQueue.global(qos: .userInitiated).async {
            do {
                try process.run()
                let errorQueue = DispatchGroup()
                let captured = CapturedData()
                errorQueue.enter()
                DispatchQueue.global(qos: .utility).async { captured.data = errors.fileHandleForReading.readDataToEndOfFile(); errorQueue.leave() }
                let timer = DispatchSource.makeTimerSource(queue: .global(qos: .utility))
                timer.schedule(deadline: .now() + timeout)
                timer.setEventHandler { if process.isRunning { process.terminate() } }
                timer.resume()
                let result = output.fileHandleForReading.readDataToEndOfFile()
                process.waitUntilExit(); errorQueue.wait(); timer.cancel()
                let error = String(data: captured.data, encoding: .utf8) ?? "Command failed"
                DispatchQueue.main.async { completion(CommandResult(status: process.terminationStatus, output: result, error: error)) }
            } catch {
                DispatchQueue.main.async { completion(CommandResult(status: 1, output: Data(), error: error.localizedDescription)) }
            }
        }
        return process
    }
}
private final class CapturedData: @unchecked Sendable { var data = Data() }

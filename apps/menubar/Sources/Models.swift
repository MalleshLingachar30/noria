import Foundation

struct QuotaWindow: Decodable {
    let name: String
    let model: String?
    let used_percent: Double?
    let resets_at: Date?
    let duration: Int64?
}
struct QuotaSnapshot: Decodable {
    let observed_at: Date
    let available: Bool
    let auth_invalid: Bool?
    let windows: [QuotaWindow]?
}
struct AccountStatus: Decodable {
    let id: String
    let email: String?
    let alias: String?
    let provider: String
    let enabled: Bool
    let inflight: Int?
    let snapshot: QuotaSnapshot?
    var title: String { if let alias, !alias.isEmpty { return alias }; if let email, !email.isEmpty { return email }; return "Sign-in needed" }
}
struct Preferences: Decodable {
    let start_at_login: Bool
    let menubar_at_login: Bool
    let auto_update: Bool
}
struct ClientRoot: Decodable { let provider: String; let root: String }
struct ClientIntegration: Decodable {
    let provider: String
    let path: String
    let active: Bool
    let healthy: Bool
}
struct Dashboard: Decodable {
    let schema_version: Int
    let version: String
    let running: Bool
    let accounts: [AccountStatus]
    let preferences: Preferences
    let client_roots: [ClientRoot]
    let integrations: [ClientIntegration]
}
struct UpdateResult: Decodable {
    let current: String
    let latest: String
    let available: Bool
    let updated: Bool
    let deferred: Bool
    let message: String
}
func decodeJSON<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .custom { decoder in
        let value = try decoder.singleValueContainer().decode(String.self)
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: value) { return date }
        formatter.formatOptions = [.withInternetDateTime]
        if let date = formatter.date(from: value) { return date }
        throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Invalid timestamp"))
    }
    return try decoder.decode(type, from: data)
}
func windowText(_ window: QuotaWindow?, title: String) -> String {
    guard let window else { return "\(title)    Usage unavailable" }
    let left = window.used_percent.map { String(format: "%.0f%% left", max(0, min(100, 100 - $0))) } ?? "Unknown"
    guard let reset = window.resets_at else { return "\(title)    \(left) · reset unknown" }
    let seconds = reset.timeIntervalSinceNow
    if seconds <= 0 { return "\(title)    \(left) · refreshing reset" }
    let minutes = Int(ceil(seconds / 60))
    let countdown: String
    if minutes >= 1440 { countdown = "\(minutes / 1440)d \((minutes % 1440) / 60)h" }
    else if minutes >= 60 { countdown = "\(minutes / 60)h \(minutes % 60)m" }
    else { countdown = "\(minutes)m" }
    return "\(title)    \(left) · resets in \(countdown)"
}

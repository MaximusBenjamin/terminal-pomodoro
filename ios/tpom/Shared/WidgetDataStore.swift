import Foundation

struct WidgetSnapshot: Codable {
    let dailyHours: [DailyEntry]
    let weekByHabit: [HabitWeek]
    let todayHours: Double
    let weekHours: Double
    let allTimeHours: Double
    let updatedAt: Date
}

struct WidgetAuthToken: Codable {
    let accessToken: String
    let refreshToken: String
}

enum WidgetDataStore {
    static let suiteName = "group.com.maximus.tpom"
    private static let key = "widgetSnapshot"
    private static let tokenKey = "widgetAuthToken"

    static func save(_ snapshot: WidgetSnapshot) {
        guard let data = try? JSONEncoder().encode(snapshot) else { return }
        UserDefaults(suiteName: suiteName)?.set(data, forKey: key)
    }

    static func load() -> WidgetSnapshot? {
        guard let data = UserDefaults(suiteName: suiteName)?.data(forKey: key) else { return nil }
        return try? JSONDecoder().decode(WidgetSnapshot.self, from: data)
    }

    static func saveToken(_ token: WidgetAuthToken) {
        guard let data = try? JSONEncoder().encode(token) else { return }
        UserDefaults(suiteName: suiteName)?.set(data, forKey: tokenKey)
    }

    static func loadToken() -> WidgetAuthToken? {
        guard let data = UserDefaults(suiteName: suiteName)?.data(forKey: tokenKey) else { return nil }
        return try? JSONDecoder().decode(WidgetAuthToken.self, from: data)
    }

    static func clear() {
        let defaults = UserDefaults(suiteName: suiteName)
        defaults?.removeObject(forKey: key)
        defaults?.removeObject(forKey: tokenKey)
    }
}

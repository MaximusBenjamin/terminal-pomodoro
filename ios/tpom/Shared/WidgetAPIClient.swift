import Foundation

/// Lightweight Supabase client for widget extensions.
/// Uses URLSession directly — no Supabase SDK dependency.
enum WidgetAPIClient {
    private static let supabaseURL = "https://dqnvsgtksqhbrmqchlds.supabase.co"
    private static let supabaseKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImRxbnZzZ3Rrc3FoYnJtcWNobGRzIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzUxNjQxNDMsImV4cCI6MjA5MDc0MDE0M30.ew7OMJlZQDdpi1d7Mfe6s7kcA6wuNtAIZxFGNdVBxjw"

    /// Fetch fresh data from Supabase and build a WidgetSnapshot.
    /// Returns nil if auth token is missing or requests fail.
    static func fetchSnapshot() async -> WidgetSnapshot? {
        guard var token = WidgetDataStore.loadToken() else { return nil }

        // Try fetching with current token
        if let snapshot = await fetchWithToken(token.accessToken) {
            return snapshot
        }

        // Token might be expired — try refreshing
        guard let refreshed = await refreshToken(token.refreshToken) else { return nil }
        token = refreshed
        WidgetDataStore.saveToken(token)

        return await fetchWithToken(token.accessToken)
    }

    private static func fetchWithToken(_ accessToken: String) async -> WidgetSnapshot? {
        async let habitsResult = fetchJSON([Habit].self, path: "/rest/v1/habits?archived=eq.false&order=name", token: accessToken)
        async let sessionsResult = fetchJSON([Session].self, path: "/rest/v1/sessions?order=start_time.desc", token: accessToken)

        guard let habits = await habitsResult, let sessions = await sessionsResult else { return nil }

        return buildSnapshot(habits: habits, sessions: sessions)
    }

    private static func fetchJSON<T: Decodable>(_ type: T.Type, path: String, token: String) async -> T? {
        guard let url = URL(string: supabaseURL + path) else { return nil }
        var request = URLRequest(url: url)
        request.setValue(supabaseKey, forHTTPHeaderField: "apikey")
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        guard let (data, response) = try? await URLSession.shared.data(for: request),
              let http = response as? HTTPURLResponse,
              http.statusCode == 200 else {
            return nil
        }

        let decoder = JSONDecoder()
        return try? decoder.decode(type, from: data)
    }

    private static func refreshToken(_ refreshToken: String) async -> WidgetAuthToken? {
        guard let url = URL(string: supabaseURL + "/auth/v1/token?grant_type=refresh_token") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue(supabaseKey, forHTTPHeaderField: "apikey")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try? JSONEncoder().encode(["refresh_token": refreshToken])

        guard let (data, response) = try? await URLSession.shared.data(for: request),
              let http = response as? HTTPURLResponse,
              http.statusCode == 200 else {
            return nil
        }

        struct AuthResponse: Decodable {
            let access_token: String
            let refresh_token: String
        }
        guard let authResp = try? JSONDecoder().decode(AuthResponse.self, from: data) else { return nil }
        return WidgetAuthToken(accessToken: authResp.access_token, refreshToken: authResp.refresh_token)
    }

    // MARK: - Build snapshot from raw data

    private static func buildSnapshot(habits: [Habit], sessions: [Session]) -> WidgetSnapshot {
        let calendar = Calendar.current
        let now = Date()

        // Today hours (4AM boundary)
        let todayComp = effectiveDate(now)
        let todayHours = sessions
            .filter { effectiveDate(parseDate($0.startTime)) == todayComp }
            .reduce(0.0) { $0 + Double($1.actualSeconds) } / 3600.0

        // Week hours
        let shifted = calendar.date(byAdding: .hour, value: -4, to: now)!
        let weekday = calendar.component(.weekday, from: shifted)
        let daysFromMonday = (weekday == 1) ? 6 : weekday - 2
        let mondayShifted = calendar.date(byAdding: .day, value: -daysFromMonday, to: shifted)!
        let mondayStart = calendar.startOfDay(for: mondayShifted)
        let mondayCutoff = calendar.date(byAdding: .hour, value: 4, to: mondayStart)!
        let weekHours = sessions
            .filter { parseDate($0.startTime) >= mondayCutoff }
            .reduce(0.0) { $0 + Double($1.actualSeconds) } / 3600.0

        // All time
        let allTimeHours = sessions.reduce(0.0) { $0 + Double($1.actualSeconds) } / 3600.0

        // Daily hours for heatmap
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        var dayMap: [String: Double] = [:]
        for session in sessions {
            let date = parseDate(session.startTime)
            let comp = effectiveDate(date)
            if let day = calendar.date(from: comp) {
                let key = formatter.string(from: day)
                dayMap[key, default: 0] += Double(session.actualSeconds) / 3600.0
            }
        }
        var dailyHours: [DailyEntry] = []
        for offset in (0..<365).reversed() {
            if let date = calendar.date(byAdding: .day, value: -offset, to: now) {
                let key = formatter.string(from: date)
                dailyHours.append(DailyEntry(date: key, hours: dayMap[key] ?? 0))
            }
        }

        // Week by habit
        let habitLookup = Dictionary(uniqueKeysWithValues: habits.map { ($0.id, $0) })
        var habitDays: [Int: [Double]] = [:]
        for session in sessions {
            let date = parseDate(session.startTime)
            guard date >= mondayCutoff else { continue }
            let shiftedDate = calendar.date(byAdding: .hour, value: -4, to: date)!
            let wd = calendar.component(.weekday, from: shiftedDate)
            let dayIndex = (wd == 1) ? 6 : wd - 2
            if habitDays[session.habitId] == nil {
                habitDays[session.habitId] = Array(repeating: 0.0, count: 7)
            }
            habitDays[session.habitId]![dayIndex] += Double(session.actualSeconds) / 3600.0
        }
        let weekByHabit = habitDays.compactMap { habitId, daily -> HabitWeek? in
            guard let habit = habitLookup[habitId] else { return nil }
            return HabitWeek(habitName: habit.name, color: habit.color, daily: daily)
        }.sorted { $0.habitName < $1.habitName }

        return WidgetSnapshot(
            dailyHours: dailyHours,
            weekByHabit: weekByHabit,
            todayHours: todayHours,
            weekHours: weekHours,
            allTimeHours: allTimeHours,
            updatedAt: now
        )
    }

    private static func effectiveDate(_ date: Date) -> DateComponents {
        let shifted = Calendar.current.date(byAdding: .hour, value: -4, to: date)!
        return Calendar.current.dateComponents([.year, .month, .day], from: shifted)
    }

    private static func parseDate(_ str: String) -> Date {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = f.date(from: str) { return d }
        f.formatOptions = [.withInternetDateTime]
        return f.date(from: str) ?? Date()
    }
}

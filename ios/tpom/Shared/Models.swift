import Foundation

struct Habit: Codable, Identifiable {
    let id: Int
    let userId: UUID?
    let name: String
    let color: String
    let archived: Bool
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case userId = "user_id"
        case name, color, archived
        case createdAt = "created_at"
    }
}

struct Session: Codable, Identifiable {
    let id: Int
    let userId: UUID?
    let habitId: Int
    let plannedMinutes: Int
    let actualSeconds: Int
    let overtimeSeconds: Int
    let completed: Bool
    let startTime: String
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case userId = "user_id"
        case habitId = "habit_id"
        case plannedMinutes = "planned_minutes"
        case actualSeconds = "actual_seconds"
        case overtimeSeconds = "overtime_seconds"
        case completed
        case startTime = "start_time"
        case createdAt = "created_at"
    }
}

// For inserts (no id, no user_id -- Supabase sets these via defaults and RLS)
struct NewHabit: Codable {
    let name: String
    let color: String
}

struct NewSession: Codable {
    let habitId: Int
    let plannedMinutes: Int
    let actualSeconds: Int
    let overtimeSeconds: Int
    let completed: Bool
    let startTime: String

    enum CodingKeys: String, CodingKey {
        case habitId = "habit_id"
        case plannedMinutes = "planned_minutes"
        case actualSeconds = "actual_seconds"
        case overtimeSeconds = "overtime_seconds"
        case completed
        case startTime = "start_time"
    }
}

// MARK: - Todo

struct Todo: Codable, Identifiable, Equatable {
    let id: Int
    let userId: UUID?
    let text: String
    let completed: Bool
    let effectiveDate: String   // YYYY-MM-DD
    let createdAt: String?
    let completedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case userId = "user_id"
        case text, completed
        case effectiveDate = "effective_date"
        case createdAt = "created_at"
        case completedAt = "completed_at"
    }
}

struct NewTodo: Codable {
    let text: String
    let effectiveDate: String  // YYYY-MM-DD

    enum CodingKeys: String, CodingKey {
        case text
        case effectiveDate = "effective_date"
    }
}

// For session list display (joined with habit)
struct SessionWithHabit: Identifiable {
    let session: Session
    let habit: Habit?
    var id: Int { session.id }
}

// Stats display models used by HeatmapView, WeeklyChartView, StatsView
struct DailyEntry: Identifiable, Codable {
    let date: String
    let hours: Double
    var id: String { date }

    enum CodingKeys: String, CodingKey {
        case date, hours
    }
}

struct HabitWeek: Identifiable, Codable {
    let habitName: String
    let color: String
    let daily: [Double] // 7 elements, Mon-Sun
    var id: String { habitName }

    enum CodingKeys: String, CodingKey {
        case habitName, color, daily
    }
}

struct HabitToday: Identifiable {
    let habitName: String
    let color: String
    let hours: Double
    var id: String { habitName }
}

struct StreakResult {
    let currentStreak: Int
    let leewayUsedWeek: Int
    let leewayRemaining: Int
    let leewayPerWeek: Int
}

// Persists user settings in shared UserDefaults
enum AppSettings {
    private static let suiteName = "group.com.maximus.tpom"
    private static let leewayKey = "leeway_days_per_week"

    static var leewayDaysPerWeek: Int {
        get { UserDefaults(suiteName: suiteName)?.integer(forKey: leewayKey) ?? 0 }
        set { UserDefaults(suiteName: suiteName)?.set(newValue, forKey: leewayKey) }
    }
}

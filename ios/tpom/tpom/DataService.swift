import Foundation
import Observation
import Supabase
import WidgetKit

@Observable
class DataService {
    var habits: [Habit] = []
    var sessions: [Session] = []
    var isLoading = false
    var error: String?
    private var realtimeTask: Task<Void, Never>?

    // MARK: - Fetch

    func fetchHabits() async {
        do {
            habits = try await supabase.from("habits")
                .select()
                .eq("archived", value: false)
                .order("name")
                .execute()
                .value
        } catch {
            self.error = friendlyError(error)
        }
    }

    func fetchSessions() async {
        do {
            sessions = try await supabase.from("sessions")
                .select()
                .order("start_time", ascending: false)
                .execute()
                .value
        } catch {
            self.error = friendlyError(error)
        }
    }

    func fetchAll() async {
        isLoading = true
        await fetchHabits()
        await fetchSessions()
        isLoading = false
        writeWidgetSnapshot()
    }

    // MARK: - Habits

    func addHabit(name: String, color: String) async {
        do {
            try await supabase.from("habits")
                .insert(NewHabit(name: name, color: color))
                .execute()
            await fetchHabits()
            writeWidgetSnapshot()
        } catch {
            self.error = friendlyError(error)
        }
    }

    func deleteHabit(id: Int) async {
        do {
            try await supabase.from("habits")
                .delete()
                .eq("id", value: id)
                .execute()
            await fetchHabits()
            writeWidgetSnapshot()
        } catch {
            self.error = friendlyError(error)
        }
    }

    // MARK: - Sessions

    func createSession(_ session: NewSession) async {
        do {
            try await supabase.from("sessions")
                .insert(session)
                .execute()
            await fetchSessions()
            writeWidgetSnapshot()
        } catch {
            self.error = friendlyError(error)
        }
    }

    func deleteSession(id: Int) async {
        do {
            try await supabase.from("sessions")
                .delete()
                .eq("id", value: id)
                .execute()
            await fetchSessions()
            writeWidgetSnapshot()
        } catch {
            self.error = friendlyError(error)
        }
    }

    func updateSession(id: Int, session: NewSession) async {
        do {
            try await supabase.from("sessions")
                .update(session)
                .eq("id", value: id)
                .execute()
            await fetchSessions()
            writeWidgetSnapshot()
        } catch {
            self.error = friendlyError(error)
        }
    }

    // MARK: - Realtime

    func startRealtime() async {
        // Don't start twice
        guard realtimeTask == nil else { return }

        let channel = supabase.realtimeV2.channel("db-changes")

        let sessionChanges = channel.postgresChange(AnyAction.self, table: "sessions")
        let habitChanges = channel.postgresChange(AnyAction.self, table: "habits")

        await channel.subscribe()

        realtimeTask = Task {
            await withTaskGroup(of: Void.self) { group in
                group.addTask {
                    for await _ in sessionChanges {
                        await self.fetchSessions()
                        self.writeWidgetSnapshot()
                    }
                }
                group.addTask {
                    for await _ in habitChanges {
                        await self.fetchHabits()
                        self.writeWidgetSnapshot()
                    }
                }
            }
        }
    }

    func stopRealtime() {
        realtimeTask?.cancel()
        realtimeTask = nil
    }

    // MARK: - Widget

    func writeWidgetSnapshot() {
        let snapshot = WidgetSnapshot(
            dailyHours: dailyHoursForYear(),
            weekByHabit: weekByHabit(),
            todayHours: todayHours(),
            weekHours: weekHours(),
            allTimeHours: allTimeHours(),
            updatedAt: Date()
        )
        WidgetDataStore.save(snapshot)
        WidgetCenter.shared.reloadAllTimelines()
    }

    // MARK: - Computed Stats (4AM day boundary, matching TUI)

    func todayHours() -> Double {
        let todayComp = effectiveDate(Date())
        return sessions
            .filter { effectiveDate(parseDate($0.startTime)) == todayComp }
            .reduce(0.0) { $0 + Double($1.actualSeconds) } / 3600.0
    }

    func weekHours() -> Double {
        let calendar = Calendar.current
        let now = Date()
        let shifted = calendar.date(byAdding: .hour, value: -4, to: now)!
        // Find the start of the current ISO week (Monday)
        let weekday = calendar.component(.weekday, from: shifted)
        let daysFromMonday = (weekday == 1) ? 6 : weekday - 2
        let mondayStart = calendar.date(byAdding: .day, value: -daysFromMonday, to: shifted)!
        let monday4am = calendar.startOfDay(for: mondayStart)

        // The actual cutoff is Monday at 4AM
        let mondayCutoff = calendar.date(byAdding: .hour, value: 4, to: monday4am)!

        return sessions
            .filter { parseDate($0.startTime) >= mondayCutoff }
            .reduce(0.0) { $0 + Double($1.actualSeconds) } / 3600.0
    }

    func allTimeHours() -> Double {
        sessions.reduce(0.0) { $0 + Double($1.actualSeconds) } / 3600.0
    }

    func dailyHoursForYear() -> [DailyEntry] {
        let calendar = Calendar.current
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        // Group sessions by effective date
        var dayMap: [String: Double] = [:]
        for session in sessions {
            let date = parseDate(session.startTime)
            let comp = effectiveDate(date)
            if let day = calendar.date(from: comp) {
                let key = formatter.string(from: day)
                dayMap[key, default: 0] += Double(session.actualSeconds) / 3600.0
            }
        }

        // Build entries for the last 365 days
        let today = Date()
        var entries: [DailyEntry] = []
        for offset in (0..<365).reversed() {
            if let date = calendar.date(byAdding: .day, value: -offset, to: today) {
                let key = formatter.string(from: date)
                entries.append(DailyEntry(date: key, hours: dayMap[key] ?? 0))
            }
        }
        return entries
    }

    func weekByHabit() -> [HabitWeek] {
        let calendar = Calendar.current
        let now = Date()
        let shifted = calendar.date(byAdding: .hour, value: -4, to: now)!
        let weekday = calendar.component(.weekday, from: shifted)
        let daysFromMonday = (weekday == 1) ? 6 : weekday - 2
        let mondayShifted = calendar.date(byAdding: .day, value: -daysFromMonday, to: shifted)!
        let mondayStart = calendar.startOfDay(for: mondayShifted)
        let mondayCutoff = calendar.date(byAdding: .hour, value: 4, to: mondayStart)!

        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        // Build a lookup from habit id to habit
        let habitLookup = Dictionary(uniqueKeysWithValues: habits.map { ($0.id, $0) })

        // Group sessions by habit, then by weekday
        var habitDays: [Int: [Double]] = [:]
        for session in sessions {
            let date = parseDate(session.startTime)
            guard date >= mondayCutoff else { continue }

            let shiftedDate = calendar.date(byAdding: .hour, value: -4, to: date)!
            let wd = calendar.component(.weekday, from: shiftedDate)
            let dayIndex = (wd == 1) ? 6 : wd - 2 // 0=Mon, 6=Sun

            if habitDays[session.habitId] == nil {
                habitDays[session.habitId] = Array(repeating: 0.0, count: 7)
            }
            habitDays[session.habitId]![dayIndex] += Double(session.actualSeconds) / 3600.0
        }

        return habitDays.compactMap { habitId, daily in
            guard let habit = habitLookup[habitId] else { return nil }
            return HabitWeek(habitName: habit.name, color: habit.color, daily: daily)
        }.sorted { $0.habitName < $1.habitName }
    }

    func todayByHabit() -> [HabitToday] {
        let todayComp = effectiveDate(Date())
        let habitLookup = Dictionary(uniqueKeysWithValues: habits.map { ($0.id, $0) })

        var habitHours: [Int: Double] = [:]
        for session in sessions {
            let comp = effectiveDate(parseDate(session.startTime))
            guard comp == todayComp else { continue }
            habitHours[session.habitId, default: 0] += Double(session.actualSeconds) / 3600.0
        }

        return habitHours.compactMap { habitId, hours in
            guard let habit = habitLookup[habitId] else { return nil }
            return HabitToday(habitName: habit.name, color: habit.color, hours: hours)
        }.sorted { $0.hours > $1.hours }
    }

    func sessionsWithHabits() -> [SessionWithHabit] {
        let habitLookup = Dictionary(uniqueKeysWithValues: habits.map { ($0.id, $0) })
        return sessions.map { session in
            SessionWithHabit(session: session, habit: habitLookup[session.habitId])
        }
    }

    // MARK: - Error Handling

    private func friendlyError(_ error: Error) -> String {
        let desc = error.localizedDescription.lowercased()
        if desc.contains("network") || desc.contains("not connected") || desc.contains("timed out") || desc.contains("offline") {
            return "No internet connection. Please check your network."
        }
        if desc.contains("401") || desc.contains("unauthorized") || desc.contains("jwt") || desc.contains("token") {
            return "Session expired. Please sign in again."
        }
        if desc.contains("403") || desc.contains("forbidden") {
            return "Permission denied."
        }
        if desc.contains("duplicate") || desc.contains("unique") {
            return "This item already exists."
        }
        return "Something went wrong. Please try again."
    }

    func clearData() {
        habits = []
        sessions = []
        error = nil
        stopRealtime()
        WidgetDataStore.clear()
    }

    // MARK: - Helpers

    private func effectiveDate(_ date: Date) -> DateComponents {
        let shifted = Calendar.current.date(byAdding: .hour, value: -4, to: date)!
        return Calendar.current.dateComponents([.year, .month, .day], from: shifted)
    }

    private func parseDate(_ str: String) -> Date {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = f.date(from: str) { return d }
        // Try without fractional seconds
        f.formatOptions = [.withInternetDateTime]
        return f.date(from: str) ?? Date()
    }
}

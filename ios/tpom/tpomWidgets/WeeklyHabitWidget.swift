import WidgetKit
import SwiftUI

// MARK: - Timeline

private struct WeeklyHabitEntry: TimelineEntry {
    let date: Date
    let habits: [HabitWeek]
    let hasData: Bool
}

private struct WeeklyHabitProvider: TimelineProvider {
    func placeholder(in context: Context) -> WeeklyHabitEntry {
        WeeklyHabitEntry(date: .now, habits: Self.sampleHabits(), hasData: true)
    }

    func getSnapshot(in context: Context, completion: @escaping (WeeklyHabitEntry) -> Void) {
        if let snap = WidgetDataStore.load(), !snap.weekByHabit.isEmpty {
            completion(WeeklyHabitEntry(date: snap.updatedAt, habits: snap.weekByHabit, hasData: true))
        } else {
            completion(placeholder(in: context))
        }
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<WeeklyHabitEntry>) -> Void) {
        Task {
            if let snap = await WidgetAPIClient.fetchSnapshot() {
                WidgetDataStore.save(snap)
                let entry = WeeklyHabitEntry(date: snap.updatedAt, habits: snap.weekByHabit, hasData: true)
                let nextRefresh = Date().addingTimeInterval(15 * 60)
                completion(Timeline(entries: [entry], policy: .after(nextRefresh)))
                return
            }
            let entry: WeeklyHabitEntry
            if let snap = WidgetDataStore.load(), !snap.weekByHabit.isEmpty {
                entry = WeeklyHabitEntry(date: snap.updatedAt, habits: snap.weekByHabit, hasData: true)
            } else {
                entry = WeeklyHabitEntry(date: .now, habits: [], hasData: false)
            }
            let nextRefresh = Date().addingTimeInterval(15 * 60)
            completion(Timeline(entries: [entry], policy: .after(nextRefresh)))
        }
    }

    static func sampleHabits() -> [HabitWeek] {
        [
            HabitWeek(habitName: "Deep Work", color: "#7aa2f7", daily: [2.0, 1.5, 3.0, 2.5, 1.0, 0.5, 0.0]),
            HabitWeek(habitName: "Reading", color: "#9ece6a", daily: [0.5, 1.0, 0.5, 0.75, 0.5, 1.5, 1.0]),
            HabitWeek(habitName: "Exercise", color: "#e0af68", daily: [1.0, 0.0, 1.0, 0.0, 1.0, 0.5, 0.0]),
        ]
    }
}

// MARK: - Grid View

private struct HabitGridView: View {
    let habits: [HabitWeek]

    private let dayLabels = ["M", "T", "W", "T", "F", "S", "S"]
    private let labelWidth: CGFloat = 90
    private let cellSize: CGFloat = 14
    private let cellSpacing: CGFloat = 2
    private let cornerRadius: CGFloat = 3

    var body: some View {
        GeometryReader { geo in
            let gridWidth = geo.size.width - labelWidth
            let totalCellWidth = cellSize * 7
            let gap = (gridWidth - totalCellWidth) / 7

            // Compute vertical spacing to fill height
            let habitCount = CGFloat(min(habits.count, 5))
            let headerRow: CGFloat = 16
            let usedHeight = headerRow + habitCount * cellSize
            let rowSpacing = habitCount > 0 ? (geo.size.height - usedHeight) / (habitCount + 1) : 6

            VStack(alignment: .leading, spacing: rowSpacing) {
                // Day header row
                HStack(spacing: 0) {
                    Color.clear
                        .frame(width: labelWidth, height: 1)
                    let todayIndex = currentDayIndex()
                    HStack(spacing: 0) {
                        ForEach(0..<7, id: \.self) { i in
                            Text(dayLabels[i])
                                .font(.system(size: 10, weight: .semibold))
                                .foregroundStyle(i == todayIndex ? Theme.accent : Theme.muted)
                                .frame(width: cellSize + gap)
                        }
                    }
                }

                // Habit rows
                ForEach(habits.prefix(5)) { habit in
                    let habitColor = Color(hex: habit.color)
                    let maxHours = habit.daily.max() ?? 0

                    HStack(spacing: 0) {
                        HStack(spacing: 5) {
                            Circle()
                                .fill(habitColor)
                                .frame(width: 6, height: 6)
                            Text(habit.habitName)
                                .font(.system(size: 12, weight: .medium))
                                .foregroundStyle(Theme.fg)
                                .lineLimit(1)
                        }
                        .frame(width: labelWidth, alignment: .leading)

                        HStack(spacing: 0) {
                            ForEach(0..<7, id: \.self) { dayIndex in
                                let hours = habit.daily.count > dayIndex ? habit.daily[dayIndex] : 0
                                RoundedRectangle(cornerRadius: cornerRadius)
                                    .fill(colorForHabit(hours: hours, maxHours: maxHours, color: habitColor))
                                    .frame(width: cellSize, height: cellSize)
                                    .frame(width: cellSize + gap)
                            }
                        }
                    }
                }
            }
        }
    }

    /// Returns 0=Mon, 1=Tue, ..., 6=Sun for the current day.
    private func currentDayIndex() -> Int {
        let wd = Calendar.current.component(.weekday, from: Date())
        // weekday: 1=Sun, 2=Mon, ..., 7=Sat → 0=Mon, ..., 6=Sun
        return wd == 1 ? 6 : wd - 2
    }

    /// Returns the habit's own color at varying opacity based on hours relative to that habit's max.
    private func colorForHabit(hours: Double, maxHours: Double, color: Color) -> Color {
        if hours == 0 { return Theme.heatmap0 }
        if maxHours == 0 { return Theme.heatmap0 }
        // Scale opacity from 0.3 (low) to 1.0 (max)
        let ratio = hours / maxHours
        let opacity = 0.3 + ratio * 0.7
        return color.opacity(opacity)
    }
}

// MARK: - Widget

struct WeeklyHabitWidget: Widget {
    let kind = "WeeklyHabitWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: WeeklyHabitProvider()) { entry in
            if entry.hasData {
                if #available(iOSApplicationExtension 17.0, *) {
                    HabitGridView(habits: entry.habits)
                        .containerBackground(Theme.bg, for: .widget)
                } else {
                    HabitGridView(habits: entry.habits)
                }
            } else {
                if #available(iOSApplicationExtension 17.0, *) {
                    NoDataView()
                        .containerBackground(Theme.bg, for: .widget)
                } else {
                    NoDataView()
                }
            }
        }
        .configurationDisplayName("Habit Grid")
        .description("Weekly hours by habit")
        .supportedFamilies([.systemMedium])
    }
}

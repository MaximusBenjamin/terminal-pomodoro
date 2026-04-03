import WidgetKit
import SwiftUI

// MARK: - Timeline

private struct WeeklyChartEntry: TimelineEntry {
    let date: Date
    let habits: [HabitWeek]
    let todayHours: Double
    let weekHours: Double
    let allTimeHours: Double
    let hasData: Bool
}

private struct WeeklyChartProvider: TimelineProvider {
    func placeholder(in context: Context) -> WeeklyChartEntry {
        WeeklyChartEntry(date: .now, habits: Self.sampleHabits(), todayHours: 1.5, weekHours: 8.2, allTimeHours: 142.7, hasData: true)
    }

    func getSnapshot(in context: Context, completion: @escaping (WeeklyChartEntry) -> Void) {
        if let snap = WidgetDataStore.load(), !snap.weekByHabit.isEmpty {
            completion(WeeklyChartEntry(date: snap.updatedAt, habits: snap.weekByHabit, todayHours: snap.todayHours, weekHours: snap.weekHours, allTimeHours: snap.allTimeHours, hasData: true))
        } else {
            completion(placeholder(in: context))
        }
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<WeeklyChartEntry>) -> Void) {
        Task {
            if let snap = await WidgetAPIClient.fetchSnapshot() {
                WidgetDataStore.save(snap)
                let entry = WeeklyChartEntry(date: snap.updatedAt, habits: snap.weekByHabit, todayHours: snap.todayHours, weekHours: snap.weekHours, allTimeHours: snap.allTimeHours, hasData: true)
                let nextRefresh = Date().addingTimeInterval(15 * 60)
                completion(Timeline(entries: [entry], policy: .after(nextRefresh)))
                return
            }
            let entry: WeeklyChartEntry
            if let snap = WidgetDataStore.load(), !snap.weekByHabit.isEmpty {
                entry = WeeklyChartEntry(date: snap.updatedAt, habits: snap.weekByHabit, todayHours: snap.todayHours, weekHours: snap.weekHours, allTimeHours: snap.allTimeHours, hasData: true)
            } else {
                entry = WeeklyChartEntry(date: .now, habits: [], todayHours: 0, weekHours: 0, allTimeHours: 0, hasData: false)
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

// MARK: - Format Helper

private func chartFmtDuration(_ hours: Double) -> String {
    let totalMinutes = hours * 60
    if totalMinutes < 1 { return "0m" }
    if hours < 1 { return "\(Int(totalMinutes))m" }
    return String(format: "%.1fh", hours)
}

// MARK: - Chart View

private struct WeeklyChartContentView: View {
    let habits: [HabitWeek]
    let todayHours: Double
    let weekHours: Double
    let allTimeHours: Double
    let isLarge: Bool

    private let dayLabels = ["M", "T", "W", "T", "F", "S", "S"]

    var body: some View {
        let maxTotal = calculateMaxTotal()
        let yMax = maxTotal > 0 ? ceil(maxTotal) : 1

        VStack(spacing: 8) {
            // Summary stats row
            HStack(spacing: 0) {
                statItem(label: "Today", value: todayHours)
                statDivider
                statItem(label: "Week", value: weekHours)
                statDivider
                statItem(label: "All Time", value: allTimeHours)
            }

            // Chart + legend
            HStack(spacing: 10) {
                // Y-axis + bars
                HStack(alignment: .bottom, spacing: 0) {
                    // Y-axis labels
                    GeometryReader { geo in
                        let barArea = geo.size.height - 22 // minus day label + spacing
                        VStack(alignment: .trailing, spacing: 0) {
                            Text(chartFmtDuration(yMax))
                                .font(.system(size: 8, weight: .medium, design: .monospaced))
                                .foregroundStyle(Theme.muted)
                            Spacer(minLength: 0)
                            Text(chartFmtDuration(yMax / 2))
                                .font(.system(size: 8, weight: .medium, design: .monospaced))
                                .foregroundStyle(Theme.muted)
                            Spacer(minLength: 0)
                            Text("0")
                                .font(.system(size: 8, weight: .medium, design: .monospaced))
                                .foregroundStyle(Theme.muted)
                                .padding(.bottom, 22)
                        }
                        .frame(maxHeight: .infinity)
                    }
                    .frame(width: 28)

                    // Bars
                    HStack(alignment: .bottom, spacing: isLarge ? 8 : 5) {
                        ForEach(0..<7, id: \.self) { dayIndex in
                            dayColumn(dayIndex: dayIndex, maxTotal: yMax)
                        }
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                }

                legendView
            }
        }
    }

    private func statItem(label: String, value: Double) -> some View {
        VStack(spacing: 2) {
            Text(label)
                .font(.system(size: 9, weight: .medium, design: .monospaced))
                .foregroundStyle(Theme.muted)
            Text(chartFmtDuration(value))
                .font(.system(size: 15, weight: .bold, design: .monospaced))
                .foregroundStyle(Theme.accent)
        }
        .frame(maxWidth: .infinity)
    }

    private var statDivider: some View {
        Rectangle()
            .fill(Theme.border)
            .frame(width: 1, height: 24)
    }

    private func dayColumn(dayIndex: Int, maxTotal: Double) -> some View {
        return VStack(spacing: 6) {
            // Stacked bar
            GeometryReader { geo in
                VStack(spacing: 0) {
                    Spacer(minLength: 0)
                    ForEach(habits.reversed()) { habit in
                        let hours = habit.daily.count > dayIndex ? habit.daily[dayIndex] : 0
                        if hours > 0 && maxTotal > 0 {
                            let segmentHeight = (hours / maxTotal) * geo.size.height
                            RoundedRectangle(cornerRadius: 2)
                                .fill(Color(hex: habit.color))
                                .frame(height: max(segmentHeight, 2))
                        }
                    }
                }
                .frame(maxWidth: .infinity)
                .clipShape(RoundedRectangle(cornerRadius: 4))
            }

            // Day label
            Text(dayLabels[dayIndex])
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(Theme.muted)
        }
        .frame(maxWidth: .infinity)
    }

    private var legendView: some View {
        let visibleHabits = habits.filter { $0.daily.contains(where: { $0 > 0 }) }

        return VStack(alignment: .leading, spacing: 8) {
            ForEach(visibleHabits.prefix(5)) { habit in
                HStack(spacing: 5) {
                    Circle()
                        .fill(Color(hex: habit.color))
                        .frame(width: 7, height: 7)
                    Text(habit.habitName)
                        .font(.system(size: 10))
                        .foregroundStyle(Theme.fg)
                        .lineLimit(1)
                }
            }
        }
        .frame(width: 90)
    }

    // MARK: - Calculations

    private func totalForDay(_ dayIndex: Int) -> Double {
        habits.reduce(0) { sum, habit in
            let hours = habit.daily.count > dayIndex ? habit.daily[dayIndex] : 0
            return sum + hours
        }
    }

    private func calculateMaxTotal() -> Double {
        var maxVal = 0.0
        for dayIndex in 0..<7 {
            maxVal = max(maxVal, totalForDay(dayIndex))
        }
        return maxVal
    }
}

// MARK: - Widget

struct WeeklyChartWidget: Widget {
    let kind = "WeeklyChartWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: WeeklyChartProvider()) { entry in
            if entry.hasData {
                if #available(iOSApplicationExtension 17.0, *) {
                    WeeklyChartContentView(habits: entry.habits, todayHours: entry.todayHours, weekHours: entry.weekHours, allTimeHours: entry.allTimeHours, isLarge: false)
                        .containerBackground(Theme.bg, for: .widget)
                } else {
                    WeeklyChartContentView(habits: entry.habits, todayHours: entry.todayHours, weekHours: entry.weekHours, allTimeHours: entry.allTimeHours, isLarge: false)
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
        .configurationDisplayName("Weekly Chart")
        .description("This week's study by category")
        .supportedFamilies([.systemMedium, .systemLarge])
    }
}

import WidgetKit
import SwiftUI

// MARK: - Timeline

private struct HeatmapEntry: TimelineEntry {
    let date: Date
    let dailyEntries: [DailyEntry]
    let hasData: Bool
}

private struct HeatmapProvider: TimelineProvider {
    func placeholder(in context: Context) -> HeatmapEntry {
        HeatmapEntry(date: .now, dailyEntries: Self.sampleEntries(), hasData: true)
    }

    func getSnapshot(in context: Context, completion: @escaping (HeatmapEntry) -> Void) {
        if let snap = WidgetDataStore.load() {
            completion(HeatmapEntry(date: snap.updatedAt, dailyEntries: snap.dailyHours, hasData: true))
        } else {
            completion(placeholder(in: context))
        }
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<HeatmapEntry>) -> Void) {
        Task {
            // Try fetching fresh data from Supabase
            if let snap = await WidgetAPIClient.fetchSnapshot() {
                WidgetDataStore.save(snap)
                let entry = HeatmapEntry(date: snap.updatedAt, dailyEntries: snap.dailyHours, hasData: true)
                let nextRefresh = Date().addingTimeInterval(15 * 60)
                completion(Timeline(entries: [entry], policy: .after(nextRefresh)))
                return
            }
            // Fall back to cached data
            let entry: HeatmapEntry
            if let snap = WidgetDataStore.load() {
                entry = HeatmapEntry(date: snap.updatedAt, dailyEntries: snap.dailyHours, hasData: true)
            } else {
                entry = HeatmapEntry(date: .now, dailyEntries: [], hasData: false)
            }
            let nextRefresh = Date().addingTimeInterval(15 * 60)
            completion(Timeline(entries: [entry], policy: .after(nextRefresh)))
        }
    }

    static func sampleEntries() -> [DailyEntry] {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let calendar = Calendar.current
        let today = Date()
        return (0..<365).map { offset in
            let date = calendar.date(byAdding: .day, value: -offset, to: today)!
            return DailyEntry(date: formatter.string(from: date), hours: Double.random(in: 0...3))
        }
    }
}

// MARK: - Heatmap Grid View

private struct HeatmapGridView: View {
    let entries: [DailyEntry]
    let isLarge: Bool

    private let cornerRadius: CGFloat = 2
    private let monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun",
                               "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]

    struct WeekData {
        let hours: [Double]       // 7 values, Mon-Sun
        let mondayDate: Date      // the Monday of this week
    }

    var body: some View {
        let lookup = buildLookup()

        if isLarge {
            let weeks = buildWeeks(lookup: lookup, days: 364)
            let firstHalf = Array(weeks.prefix(26))
            let secondHalf = Array(weeks.suffix(26))

            GeometryReader { geo in
                let halfHeight = (geo.size.height - 6) / 2
                VStack(spacing: 6) {
                    halfGrid(weeks: firstHalf, width: geo.size.width, height: halfHeight)
                    halfGrid(weeks: secondHalf, width: geo.size.width, height: halfHeight)
                }
            }
        } else {
            let weeks = buildWeeks(lookup: lookup, days: 182)
            GeometryReader { geo in
                halfGrid(weeks: weeks, width: geo.size.width, height: geo.size.height)
            }
        }
    }

    private func halfGrid(weeks: [WeekData], width: CGFloat, height: CGFloat) -> some View {
        let labelWidth: CGFloat = 14
        let monthHeight: CGFloat = 16
        let monthGap: CGFloat = 4
        let gridWidth = width - labelWidth - 4
        let gridHeight = height - monthHeight - monthGap
        let cols = CGFloat(weeks.count)

        let cellFromW = cols > 0 ? (gridWidth - (cols - 1) * 2) / cols : 0
        let cellFromH = (gridHeight - 6 * 2) / 7
        let cellSize = min(cellFromW, cellFromH)

        let hGap = cols > 1 ? (gridWidth - cols * cellSize) / (cols - 1) : 2
        let vGap = (gridHeight - 7 * cellSize) / 6

        let dayLabels = ["M", "", "W", "", "F", "", ""]
        let labels = monthLabels(weeks: weeks)

        return VStack(alignment: .leading, spacing: monthGap) {
            // Month labels
            HStack(spacing: 0) {
                Spacer().frame(width: labelWidth + 4)
                ForEach(Array(labels.enumerated()), id: \.offset) { _, label in
                    Text(label)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(Theme.fg)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .frame(height: monthHeight)

            HStack(alignment: .top, spacing: 0) {
                VStack(spacing: 0) {
                    ForEach(0..<7, id: \.self) { row in
                        Text(dayLabels[row])
                            .font(.system(size: 8, weight: .medium))
                            .foregroundStyle(Theme.muted)
                            .frame(width: labelWidth, height: cellSize + (row < 6 ? vGap : 0))
                    }
                }

                Spacer().frame(width: 4)

                HStack(spacing: hGap) {
                    ForEach(Array(weeks.enumerated()), id: \.offset) { _, week in
                        VStack(spacing: vGap) {
                            ForEach(0..<7, id: \.self) { dayIndex in
                                let hours = dayIndex < week.hours.count ? week.hours[dayIndex] : 0
                                RoundedRectangle(cornerRadius: cornerRadius)
                                    .fill(colorForHours(hours))
                                    .frame(width: cellSize, height: cellSize)
                            }
                        }
                    }
                }
            }
        }
    }

    private func monthLabels(weeks: [WeekData]) -> [String] {
        let calendar = Calendar.current
        let currentYear = calendar.component(.year, from: Date())
        var labels: [String] = []
        var lastMonth = -1

        for week in weeks {
            let year = calendar.component(.year, from: week.mondayDate)
            if year < currentYear { continue }

            let month = calendar.component(.month, from: week.mondayDate)
            if month != lastMonth {
                labels.append(monthNames[month - 1])
                lastMonth = month
            }
        }
        return labels
    }

    private func buildLookup() -> [String: Double] {
        var dict: [String: Double] = [:]
        for entry in entries {
            dict[entry.date] = entry.hours
        }
        return dict
    }

    private func buildWeeks(lookup: [String: Double], days: Int) -> [WeekData] {
        let calendar = Calendar.current
        let today = Date()
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        // Start from Jan 1, align to Monday
        let year = calendar.component(.year, from: today)
        let jan1 = calendar.date(from: DateComponents(year: year, month: 1, day: 1))!
        let jan1Wd = calendar.component(.weekday, from: jan1)
        let daysBack = jan1Wd == 1 ? 6 : jan1Wd - 2
        let startMonday = calendar.date(byAdding: .day, value: -daysBack, to: jan1)!

        let totalWeeks = days / 7

        var weeks: [WeekData] = []
        var current = startMonday

        for _ in 0..<totalWeeks {
            var hours: [Double] = []
            for dayOffset in 0..<7 {
                let date = calendar.date(byAdding: .day, value: dayOffset, to: current)!
                let dateStr = formatter.string(from: date)
                hours.append(lookup[dateStr] ?? 0)
            }
            weeks.append(WeekData(hours: hours, mondayDate: current))
            current = calendar.date(byAdding: .day, value: 7, to: current)!
        }

        return weeks
    }

    private func colorForHours(_ hours: Double) -> Color {
        if hours == 0 { return Theme.heatmap0 }
        if hours < 0.5 { return Theme.heatmap1 }
        if hours < 1.0 { return Theme.heatmap2 }
        if hours < 2.0 { return Theme.heatmap3 }
        return Theme.heatmap4
    }
}

// MARK: - Widget

private struct HeatmapEntryView: View {
    @Environment(\.widgetFamily) var family
    let entry: HeatmapEntry

    var body: some View {
        if entry.hasData {
            if #available(iOSApplicationExtension 17.0, *) {
                HeatmapGridView(entries: entry.dailyEntries, isLarge: family == .systemLarge)
                    .containerBackground(Theme.bg, for: .widget)
            } else {
                HeatmapGridView(entries: entry.dailyEntries, isLarge: family == .systemLarge)
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
}

struct HeatmapWidget: Widget {
    let kind = "HeatmapWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: HeatmapProvider()) { entry in
            HeatmapEntryView(entry: entry)
        }
        .configurationDisplayName("Activity")
        .description("Study heatmap")
        .supportedFamilies([.systemMedium, .systemLarge])
    }
}

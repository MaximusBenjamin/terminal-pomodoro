import WidgetKit
import SwiftUI

// MARK: - Timeline

private struct SummaryEntry: TimelineEntry {
    let date: Date
    let todayHours: Double
    let weekHours: Double
    let allTimeHours: Double
    let hasData: Bool
}

private struct SummaryProvider: TimelineProvider {
    func placeholder(in context: Context) -> SummaryEntry {
        SummaryEntry(date: .now, todayHours: 2.5, weekHours: 14.3, allTimeHours: 142.7, hasData: true)
    }

    func getSnapshot(in context: Context, completion: @escaping (SummaryEntry) -> Void) {
        if let snap = WidgetDataStore.load() {
            completion(SummaryEntry(
                date: snap.updatedAt,
                todayHours: snap.todayHours,
                weekHours: snap.weekHours,
                allTimeHours: snap.allTimeHours,
                hasData: true
            ))
        } else {
            completion(placeholder(in: context))
        }
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<SummaryEntry>) -> Void) {
        Task {
            if let snap = await WidgetAPIClient.fetchSnapshot() {
                WidgetDataStore.save(snap)
                let entry = SummaryEntry(date: snap.updatedAt, todayHours: snap.todayHours, weekHours: snap.weekHours, allTimeHours: snap.allTimeHours, hasData: true)
                let nextRefresh = Date().addingTimeInterval(15 * 60)
                completion(Timeline(entries: [entry], policy: .after(nextRefresh)))
                return
            }
            let entry: SummaryEntry
            if let snap = WidgetDataStore.load() {
                entry = SummaryEntry(date: snap.updatedAt, todayHours: snap.todayHours, weekHours: snap.weekHours, allTimeHours: snap.allTimeHours, hasData: true)
            } else {
                entry = SummaryEntry(date: .now, todayHours: 0, weekHours: 0, allTimeHours: 0, hasData: false)
            }
            let nextRefresh = Date().addingTimeInterval(15 * 60)
            completion(Timeline(entries: [entry], policy: .after(nextRefresh)))
        }
    }
}

// MARK: - Format Helper

private func widgetFmtDuration(_ hours: Double) -> String {
    let totalMinutes = hours * 60
    if totalMinutes < 1 { return "0m" }
    if hours < 1 { return "\(Int(totalMinutes))m" }
    return String(format: "%.1fh", hours)
}

// MARK: - Views

private struct SummarySmallView: View {
    let entry: SummaryEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Spacer()

            statRow(label: "Today", value: entry.todayHours)
            statRow(label: "Week", value: entry.weekHours)
            statRow(label: "All Time", value: entry.allTimeHours)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func statRow(label: String, value: Double) -> some View {
        HStack {
            Text(label)
                .font(.caption)
                .foregroundStyle(Theme.muted)
            Spacer()
            Text(widgetFmtDuration(value))
                .font(.callout.weight(.bold))
                .foregroundStyle(Theme.accent)
        }
    }
}

private struct SummaryMediumView: View {
    let entry: SummaryEntry

    var body: some View {
        VStack(spacing: 0) {
            Spacer()
            HStack(spacing: 0) {
                statColumn(label: "Today", value: entry.todayHours)
                divider
                statColumn(label: "This Week", value: entry.weekHours)
                divider
                statColumn(label: "All Time", value: entry.allTimeHours)
            }
        }
    }

    private func statColumn(label: String, value: Double) -> some View {
        VStack(spacing: 6) {
            Text(label)
                .font(.caption2)
                .foregroundStyle(Theme.muted)
            Text(widgetFmtDuration(value))
                .font(.title3.weight(.bold))
                .foregroundStyle(Theme.accent)
        }
        .frame(maxWidth: .infinity)
    }

    private var divider: some View {
        Rectangle()
            .fill(Theme.border)
            .frame(width: 1, height: 32)
    }
}

// MARK: - Widget

struct SummaryWidget: Widget {
    let kind = "SummaryWidget"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: SummaryProvider()) { entry in
            if entry.hasData {
                if #available(iOSApplicationExtension 17.0, *) {
                    SummaryWidgetEntryView(entry: entry)
                        .containerBackground(Theme.bg, for: .widget)
                } else {
                    SummaryWidgetEntryView(entry: entry)
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
        .configurationDisplayName("Study Stats")
        .description("Today, this week, and all-time hours")
        .supportedFamilies([.systemSmall, .systemMedium])
    }
}

private struct SummaryWidgetEntryView: View {
    @Environment(\.widgetFamily) var family
    let entry: SummaryEntry

    var body: some View {
        switch family {
        case .systemSmall:
            SummarySmallView(entry: entry)
        default:
            SummaryMediumView(entry: entry)
        }
    }
}

// MARK: - No Data Placeholder

struct NoDataView: View {
    var body: some View {
        VStack(spacing: 8) {
            Image(systemName: "clock.arrow.circlepath")
                .font(.title2)
                .foregroundStyle(Theme.muted)
            Text("Open tpom to sync")
                .font(.caption)
                .foregroundStyle(Theme.muted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

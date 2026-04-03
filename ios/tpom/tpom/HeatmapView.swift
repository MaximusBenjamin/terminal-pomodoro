import SwiftUI

struct HeatmapView: View {
    let entries: [DailyEntry]

    private let cornerRadius: CGFloat = 2
    private let dayLabels = ["M", "", "W", "", "F", "", ""]
    private let monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun",
                              "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]

    private struct WeekData: Identifiable {
        let id = UUID()
        let hours: [Double]     // 7 values, Mon-Sun
        let mondayDate: Date
    }

    var body: some View {
        let lookup = buildLookup()
        let weeks = buildWeeks(lookup: lookup, days: 364) // 52 weeks
        let firstHalf = Array(weeks.prefix(26))
        let secondHalf = Array(weeks.suffix(26))

        VStack(alignment: .leading, spacing: 16) {
            Text("Activity")
                .font(.headline)
                .foregroundStyle(Theme.accent)

            halfGrid(weeks: firstHalf)
            halfGrid(weeks: secondHalf)
        }
        .padding(16)
        .background(Theme.border.opacity(0.3))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    // MARK: - Half Grid

    @ViewBuilder
    private func halfGrid(weeks: [WeekData]) -> some View {
        let labels = monthLabels(weeks: weeks)

        GeometryReader { geo in
            let labelWidth: CGFloat = 14
            let gridWidth = geo.size.width - labelWidth - 4
            let cols = CGFloat(weeks.count)
            let minGap: CGFloat = 2
            let cellSize = (gridWidth - (cols - 1) * minGap) / cols
            let hGap = cols > 1 ? (gridWidth - cols * cellSize) / (cols - 1) : minGap

            VStack(alignment: .leading, spacing: 4) {
                // Month labels — evenly spaced
                HStack(spacing: 0) {
                    Spacer().frame(width: labelWidth + 4)
                    ForEach(Array(labels.enumerated()), id: \.offset) { _, name in
                        Text(name)
                            .font(.system(size: 10, weight: .medium))
                            .foregroundStyle(Theme.muted)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }

                HStack(alignment: .top, spacing: 0) {
                    // Day labels
                    VStack(alignment: .trailing, spacing: 0) {
                        ForEach(0..<7, id: \.self) { row in
                            Text(dayLabels[row])
                                .font(.system(size: 8))
                                .foregroundStyle(Theme.muted)
                                .frame(width: labelWidth, height: cellSize + hGap)
                        }
                    }

                    Spacer().frame(width: 4)

                    // Week columns
                    HStack(spacing: hGap) {
                        ForEach(weeks) { week in
                            VStack(spacing: hGap) {
                                ForEach(0..<7, id: \.self) { dayIndex in
                                    let hours = week.hours[dayIndex]
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
        .frame(height: 100)
    }

    // MARK: - Month Labels

    private func monthLabels(weeks: [WeekData]) -> [String] {
        let calendar = Calendar.current
        let currentYear = calendar.component(.year, from: Date())
        var labels: [String] = []
        var lastMonth = -1

        for week in weeks {
            // Skip alignment days from the previous year
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

    // MARK: - Data Processing

    private func buildLookup() -> [String: Double] {
        var dict: [String: Double] = [:]
        for entry in entries {
            dict[entry.date] = entry.hours
        }
        return dict
    }

    /// Build 52 weeks starting from Jan 1 of the current year.
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

    // MARK: - Color

    private func colorForHours(_ hours: Double) -> Color {
        if hours == 0 { return Theme.heatmap0 }
        if hours < 0.5 { return Theme.heatmap1 }
        if hours < 1.0 { return Theme.heatmap2 }
        if hours < 2.0 { return Theme.heatmap3 }
        return Theme.heatmap4
    }
}

// MARK: - Preview

#Preview {
    let sampleEntries: [DailyEntry] = {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let calendar = Calendar.current
        let today = Date()

        return (0..<365).map { offset in
            let date = calendar.date(byAdding: .day, value: -offset, to: today)!
            let dateStr = formatter.string(from: date)
            let hours = Double.random(in: 0...3)
            return DailyEntry(date: dateStr, hours: hours)
        }
    }()

    ScrollView {
        HeatmapView(entries: sampleEntries)
            .padding()
    }
    .background(Theme.bg)
}

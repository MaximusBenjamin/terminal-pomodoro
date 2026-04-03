import SwiftUI

struct SummaryCard: View {
    let today: Double
    let week: Double
    let allTime: Double

    var body: some View {
        HStack(spacing: 0) {
            statItem(label: "Today", value: today)

            divider

            statItem(label: "This Week", value: week)

            divider

            statItem(label: "All Time", value: allTime)
        }
        .padding(.vertical, 16)
        .background(Theme.border.opacity(0.3))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    private func statItem(label: String, value: Double) -> some View {
        VStack(spacing: 6) {
            Text(label)
                .font(.caption)
                .foregroundStyle(Theme.muted)

            Text(fmtDuration(value))
                .font(.title3)
                .fontWeight(.bold)
                .foregroundStyle(Theme.accent)
        }
        .frame(maxWidth: .infinity)
    }

    private var divider: some View {
        Rectangle()
            .fill(Theme.border)
            .frame(width: 1, height: 36)
    }
}

func fmtDuration(_ hours: Double) -> String {
    let totalMinutes = hours * 60
    if totalMinutes < 1 {
        return "0m"
    }
    if hours < 1 {
        return "\(Int(totalMinutes))m"
    }
    let formatted = String(format: "%.1f", hours)
    return "\(formatted)h"
}

#Preview {
    SummaryCard(today: 2.5, week: 14.3, allTime: 142.7)
        .padding()
        .background(Theme.bg)
}

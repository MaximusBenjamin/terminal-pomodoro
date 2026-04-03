import SwiftUI

struct WeeklyChartView: View {
    let habits: [HabitWeek]

    private let maxHeight: CGFloat = 180
    private let dayLabels = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]

    var body: some View {
        let maxTotal = calculateMaxTotal()

        VStack(alignment: .leading, spacing: 16) {
            Text("This Week")
                .font(.headline)
                .foregroundStyle(Theme.accent)

            // Bar chart
            HStack(alignment: .bottom, spacing: 12) {
                ForEach(0..<7, id: \.self) { dayIndex in
                    dayColumn(dayIndex: dayIndex, maxTotal: maxTotal)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.top, 8)

            // Legend
            legendView
        }
        .padding(16)
        .background(Theme.border.opacity(0.3))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    // MARK: - Day Column

    private func dayColumn(dayIndex: Int, maxTotal: Double) -> some View {
        let dayTotal = totalForDay(dayIndex)

        return VStack(spacing: 4) {
            // Hour label above bar
            Text(fmtDuration(dayTotal))
                .font(.caption2)
                .foregroundStyle(dayTotal > 0 ? Theme.fg : Theme.muted)

            // Stacked bar
            VStack(spacing: 0) {
                // Build segments in reverse so they stack bottom-up visually
                // (VStack places first item on top)
                ForEach(habits.reversed()) { habit in
                    let hours = habit.daily.count > dayIndex ? habit.daily[dayIndex] : 0
                    if hours > 0 && maxTotal > 0 {
                        let segmentHeight = (hours / maxTotal) * maxHeight
                        RoundedRectangle(cornerRadius: 3)
                            .fill(Color(hex: habit.color))
                            .frame(height: max(segmentHeight, 2))
                    }
                }
            }
            .frame(maxWidth: .infinity)
            .frame(height: maxTotal > 0 ? maxHeight : 0, alignment: .bottom)
            .clipShape(RoundedRectangle(cornerRadius: 6))

            // Day label
            Text(dayLabels[dayIndex])
                .font(.caption)
                .foregroundStyle(Theme.muted)
        }
        .frame(maxWidth: .infinity)
    }

    // MARK: - Legend

    private var legendView: some View {
        let visibleHabits = habits.filter { habit in
            habit.daily.contains(where: { $0 > 0 })
        }

        return LazyVGrid(
            columns: [
                GridItem(.adaptive(minimum: 100), spacing: 8)
            ],
            alignment: .leading,
            spacing: 6
        ) {
            ForEach(visibleHabits) { habit in
                HStack(spacing: 6) {
                    Circle()
                        .fill(Color(hex: habit.color))
                        .frame(width: 8, height: 8)

                    Text(habit.habitName)
                        .font(.caption)
                        .foregroundStyle(Theme.fg)
                        .lineLimit(1)
                }
            }
        }
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
            let total = totalForDay(dayIndex)
            maxVal = max(maxVal, total)
        }
        return maxVal
    }
}

// MARK: - Preview

#Preview {
    let sampleHabits: [HabitWeek] = [
        HabitWeek(
            habitName: "Deep Work",
            color: "#7aa2f7",
            daily: [2.0, 1.5, 3.0, 2.5, 1.0, 0.5, 0.0]
        ),
        HabitWeek(
            habitName: "Reading",
            color: "#9ece6a",
            daily: [0.5, 1.0, 0.5, 0.75, 0.5, 1.5, 1.0]
        ),
        HabitWeek(
            habitName: "Exercise",
            color: "#e0af68",
            daily: [1.0, 0.0, 1.0, 0.0, 1.0, 0.5, 0.0]
        ),
    ]

    ScrollView {
        WeeklyChartView(habits: sampleHabits)
            .padding()
    }
    .background(Theme.bg)
}

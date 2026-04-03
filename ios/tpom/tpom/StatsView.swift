import SwiftUI

struct StatsView: View {
    var dataService: DataService

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()

            ScrollView {
                VStack(spacing: 24) {
                    // Header
                    HStack {
                        Text("Stats")
                            .font(.title2.bold())
                            .foregroundStyle(Theme.accent)
                        Spacer()
                        Button {
                            Task { await dataService.fetchAll() }
                        } label: {
                            Image(systemName: "arrow.clockwise")
                                .foregroundStyle(Theme.muted)
                        }
                    }
                    .padding(.horizontal)

                    // Summary card
                    SummaryCard(
                        today: dataService.todayHours(),
                        week: dataService.weekHours(),
                        allTime: dataService.allTimeHours()
                    )
                    .padding(.horizontal)

                    // Today by habit
                    let todayHabits = dataService.todayByHabit()
                    if !todayHabits.isEmpty {
                        VStack(alignment: .leading, spacing: 12) {
                            Text("Today by Habit")
                                .font(.headline)
                                .foregroundStyle(Theme.accent)

                            ForEach(todayHabits) { item in
                                HStack {
                                    Circle()
                                        .fill(Color(hex: item.color))
                                        .frame(width: 10, height: 10)
                                    Text(item.habitName)
                                        .foregroundStyle(Theme.fg)
                                    Spacer()
                                    Text(fmtDuration(item.hours))
                                        .foregroundStyle(Theme.accent)
                                        .fontWeight(.medium)
                                }
                            }
                        }
                        .padding(16)
                        .background(Theme.border.opacity(0.3))
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .padding(.horizontal)
                    }

                    // Habit tracker grid (matches TUI)
                    let weekData = dataService.weekByHabit()
                    if !weekData.isEmpty {
                        VStack(alignment: .leading, spacing: 12) {
                            Text("Habit Tracker")
                                .font(.headline)
                                .foregroundStyle(Theme.accent)

                            // Day headers
                            HStack(spacing: 0) {
                                Color.clear.frame(width: 100, height: 1)
                                ForEach(["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"], id: \.self) { day in
                                    Text(day)
                                        .font(.system(size: 11, weight: .semibold))
                                        .foregroundStyle(Theme.muted)
                                        .frame(maxWidth: .infinity)
                                }
                            }

                            // One row per habit
                            ForEach(weekData) { habit in
                                HStack(spacing: 0) {
                                    HStack(spacing: 6) {
                                        Circle()
                                            .fill(Color(hex: habit.color))
                                            .frame(width: 8, height: 8)
                                        Text(habit.habitName)
                                            .font(.system(size: 13))
                                            .foregroundStyle(Theme.fg)
                                            .lineLimit(1)
                                    }
                                    .frame(width: 100, alignment: .leading)

                                    ForEach(0..<7, id: \.self) { dayIndex in
                                        let hours = habit.daily.count > dayIndex ? habit.daily[dayIndex] : 0
                                        RoundedRectangle(cornerRadius: 3)
                                            .fill(hours >= 5.0 / 60.0
                                                  ? Color(hex: habit.color)
                                                  : Theme.border.opacity(0.4))
                                            .frame(width: 20, height: 20)
                                            .frame(maxWidth: .infinity)
                                    }
                                }
                            }
                        }
                        .padding(16)
                        .background(Theme.border.opacity(0.3))
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .padding(.horizontal)

                        // Weekly chart
                        WeeklyChartView(habits: weekData)
                            .padding(.horizontal)
                    }

                    // Heatmap
                    HeatmapView(entries: dataService.dailyHoursForYear())
                        .padding(.horizontal)

                    // Error display
                    if let error = dataService.error {
                        Text(error)
                            .font(.caption)
                            .foregroundStyle(Theme.overtime)
                            .padding(.horizontal)
                    }

                    Spacer(minLength: 20)
                }
                .padding(.top)
            }
        }
    }
}

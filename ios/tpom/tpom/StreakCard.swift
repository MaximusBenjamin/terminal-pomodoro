import SwiftUI

struct StreakCard: View {
    let streak: StreakResult

    var body: some View {
        HStack(spacing: 0) {
            // Streak count
            VStack(spacing: 6) {
                Text("Streak")
                    .font(.caption)
                    .foregroundStyle(Theme.muted)
                HStack(alignment: .lastTextBaseline, spacing: 4) {
                    Text("\(streak.currentStreak)")
                        .font(.title3.bold())
                        .foregroundStyle(streak.currentStreak > 0 ? Theme.success : Theme.muted)
                    Text("days")
                        .font(.caption)
                        .foregroundStyle(Theme.muted)
                }
            }
            .frame(maxWidth: .infinity)

            // Divider
            Rectangle()
                .fill(Theme.border)
                .frame(width: 1, height: 36)

            // Leeway pips
            VStack(spacing: 6) {
                Text("Leeway this week")
                    .font(.caption)
                    .foregroundStyle(Theme.muted)

                if streak.leewayPerWeek == 0 {
                    Text("not set")
                        .font(.caption)
                        .foregroundStyle(Theme.muted)
                } else {
                    HStack(spacing: 6) {
                        ForEach(0..<streak.leewayPerWeek, id: \.self) { i in
                            Circle()
                                .fill(i < streak.leewayUsedWeek ? Theme.warning : Theme.success)
                                .frame(width: 10, height: 10)
                        }
                        Text("(\(streak.leewayUsedWeek)/\(streak.leewayPerWeek))")
                            .font(.caption)
                            .foregroundStyle(Theme.muted)
                    }
                }
            }
            .frame(maxWidth: .infinity)
        }
        .padding(.vertical, 16)
        .background(Theme.border.opacity(0.3))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}

import SwiftUI

struct LogView: View {
    var dataService: DataService

    @State private var showSessionForm = false
    @State private var editingSession: SessionWithHabit?

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()

            VStack(spacing: 0) {
                // Header
                HStack {
                    Text("Session Log")
                        .font(.title2.bold())
                        .foregroundStyle(Theme.accent)
                    Spacer()
                    Button {
                        editingSession = nil
                        showSessionForm = true
                    } label: {
                        Image(systemName: "plus.circle.fill")
                            .foregroundStyle(Theme.accent)
                    }
                    Button {
                        Task { await dataService.fetchSessions() }
                    } label: {
                        Image(systemName: "arrow.clockwise")
                            .foregroundStyle(Theme.muted)
                    }
                }
                .padding(.horizontal)
                .padding(.top, 8)
                .padding(.bottom, 16)

                if dataService.sessions.isEmpty {
                    Spacer()
                    VStack(spacing: 12) {
                        Image(systemName: "clock.arrow.circlepath")
                            .font(.system(size: 40))
                            .foregroundStyle(Theme.muted)
                        Text("No sessions yet")
                            .foregroundStyle(Theme.muted)
                        Text("Complete a timer to see it here")
                            .font(.caption)
                            .foregroundStyle(Theme.muted)
                    }
                    Spacer()
                } else {
                    let items = dataService.sessionsWithHabits()
                    List {
                        ForEach(items) { item in
                            sessionRow(item)
                                .listRowBackground(Theme.border.opacity(0.15))
                                .contentShape(Rectangle())
                                .onTapGesture {
                                    editingSession = item
                                    showSessionForm = true
                                }
                        }
                        .onDelete { indexSet in
                            Task {
                                for index in indexSet {
                                    let session = items[index].session
                                    await dataService.deleteSession(id: session.id)
                                }
                            }
                        }
                    }
                    .listStyle(.plain)
                    .scrollContentBackground(.hidden)
                }
            }
            .sheet(isPresented: $showSessionForm) {
                SessionFormView(
                    dataService: dataService,
                    editing: editingSession
                )
            }
        }
    }

    @ViewBuilder
    private func sessionRow(_ item: SessionWithHabit) -> some View {
        let session = item.session
        HStack(spacing: 12) {
            // Habit color dot
            Circle()
                .fill(Color(hex: item.habit?.color ?? "#7aa2f7"))
                .frame(width: 10, height: 10)

            VStack(alignment: .leading, spacing: 4) {
                Text(item.habit?.name ?? "Unknown")
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(Theme.fg)

                Text(formatSessionDate(session.startTime))
                    .font(.caption)
                    .foregroundStyle(Theme.muted)
            }

            Spacer()

            VStack(alignment: .trailing, spacing: 4) {
                Text(formatDuration(seconds: session.actualSeconds))
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(Theme.accent)

                if session.overtimeSeconds > 0 {
                    Text("+\(formatDuration(seconds: session.overtimeSeconds))")
                        .font(.caption)
                        .foregroundStyle(Theme.overtime)
                }
            }
        }
        .padding(.vertical, 4)
    }

    private func formatDuration(seconds: Int) -> String {
        let h = seconds / 3600
        let m = (seconds % 3600) / 60
        let s = seconds % 60
        if h > 0 {
            return String(format: "%dh %02dm", h, m)
        } else if m > 0 {
            return String(format: "%dm %02ds", m, s)
        } else {
            return String(format: "%ds", s)
        }
    }

    private func formatSessionDate(_ str: String) -> String {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        var date = f.date(from: str)
        if date == nil {
            f.formatOptions = [.withInternetDateTime]
            date = f.date(from: str)
        }
        guard let date else { return str }

        let display = DateFormatter()
        display.dateFormat = "MMM d, h:mm a"
        return display.string(from: date)
    }
}

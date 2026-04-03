import SwiftUI
import Combine

enum TimerState {
    case idle
    case running
    case paused
    case overtime
    case confirming
}

struct TimerView: View {
    var dataService: DataService

    @State private var timerState: TimerState = .idle
    @State private var selectedHabitId: Int?
    @State private var plannedMinutes = 25
    @State private var remainingSeconds = 25 * 60
    @State private var overtimeSeconds = 0
    @State private var elapsedSeconds = 0
    @State private var startTime = Date()
    @State private var timerCancellable: AnyCancellable?
    @State private var showSignOutAlert = false

    private var selectedHabit: Habit? {
        guard let id = selectedHabitId else {
            return dataService.habits.first
        }
        return dataService.habits.first { $0.id == id }
    }

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()

            VStack(spacing: 0) {
                // Header
                HStack {
                    Text("tpom")
                        .font(.system(.title2, design: .monospaced).bold())
                        .foregroundStyle(Theme.accent)
                    Spacer()
                    Button {
                        showSignOutAlert = true
                    } label: {
                        Image(systemName: "rectangle.portrait.and.arrow.right")
                            .foregroundStyle(Theme.muted)
                    }
                    .alert("Sign Out", isPresented: $showSignOutAlert) {
                        Button("Sign Out", role: .destructive) {
                            dataService.clearData()
                            Task { try? await supabase.auth.signOut() }
                        }
                        Button("Cancel", role: .cancel) {}
                    } message: {
                        Text("Are you sure you want to sign out?")
                    }
                }
                .padding(.horizontal)
                .padding(.top, 8)

                Spacer()

                // Habit selector
                if !dataService.habits.isEmpty && timerState == .idle {
                    Menu {
                        ForEach(dataService.habits) { habit in
                            Button {
                                selectedHabitId = habit.id
                            } label: {
                                Label(habit.name, systemImage: selectedHabitId == habit.id || (selectedHabitId == nil && habit.id == dataService.habits.first?.id) ? "checkmark.circle.fill" : "circle")
                            }
                        }
                    } label: {
                        HStack(spacing: 8) {
                            if let habit = selectedHabit {
                                Circle()
                                    .fill(Color(hex: habit.color))
                                    .frame(width: 10, height: 10)
                                Text(habit.name)
                                    .font(.system(.body, design: .monospaced).weight(.medium))
                                    .foregroundStyle(Theme.fg)
                            }
                            Image(systemName: "chevron.down")
                                .font(.caption)
                                .foregroundStyle(Theme.muted)
                        }
                        .padding(.horizontal, 20)
                        .padding(.vertical, 10)
                        .background(Theme.border.opacity(0.3))
                        .clipShape(Capsule())
                    }
                    .padding(.bottom, 20)
                } else if timerState != .idle, let habit = selectedHabit {
                    Text(habit.name)
                        .font(.system(.headline, design: .monospaced))
                        .foregroundStyle(Color(hex: habit.color))
                        .padding(.bottom, 20)
                }

                // Timer display with circular progress
                ZStack {
                    // Background ring
                    Circle()
                        .stroke(Theme.border.opacity(0.3), lineWidth: 8)
                        .frame(width: 260, height: 260)

                    // Progress ring
                    Circle()
                        .trim(from: 0, to: progressFraction)
                        .stroke(
                            timerState == .overtime ? Theme.overtime : Theme.accent,
                            style: StrokeStyle(lineWidth: 8, lineCap: .round)
                        )
                        .frame(width: 260, height: 260)
                        .rotationEffect(.degrees(-90))
                        .animation(.linear(duration: 0.5), value: progressFraction)

                    // Time display
                    VStack(spacing: 6) {
                        if timerState == .overtime {
                            Text("OVERTIME")
                                .font(.system(size: 12, weight: .bold, design: .monospaced))
                                .foregroundStyle(Theme.overtime)
                                .tracking(3)
                        }

                        Text(timeString)
                            .font(.system(size: 58, weight: .medium, design: .monospaced))
                            .foregroundStyle(timerState == .overtime ? Theme.overtime : Theme.fg)

                        if timerState == .confirming {
                            Text("save session?")
                                .font(.system(size: 13, design: .monospaced))
                                .foregroundStyle(Theme.muted)
                        } else if timerState == .paused {
                            Text("PAUSED")
                                .font(.system(size: 12, weight: .bold, design: .monospaced))
                                .foregroundStyle(Theme.warning)
                                .tracking(3)
                        }
                    }
                }
                .padding(.vertical, 20)

                // Time adjustment (idle only)
                if timerState == .idle {
                    HStack(spacing: 40) {
                        Button {
                            adjustTime(by: -5)
                        } label: {
                            Image(systemName: "minus.circle.fill")
                                .font(.title)
                                .foregroundStyle(Theme.muted)
                        }

                        Button {
                            adjustTime(by: 5)
                        } label: {
                            Image(systemName: "plus.circle.fill")
                                .font(.title)
                                .foregroundStyle(Theme.muted)
                        }
                    }
                    .padding(.bottom, 24)
                }

                Spacer()

                // Controls
                controlButtons
                    .padding(.bottom, 40)
            }
        }
    }

    // MARK: - Controls

    @ViewBuilder
    private var controlButtons: some View {
        switch timerState {
        case .idle:
            Button {
                startTimer()
            } label: {
                Label("Start", systemImage: "play.fill")
                    .font(.headline)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(Theme.accent)
                    .foregroundStyle(Theme.bg)
                    .clipShape(RoundedRectangle(cornerRadius: 14))
            }
            .disabled(selectedHabit == nil)
            .padding(.horizontal, 48)

        case .running, .overtime:
            HStack(spacing: 16) {
                Button {
                    pauseTimer()
                } label: {
                    Label("Pause", systemImage: "pause.fill")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(Theme.warning)
                        .foregroundStyle(Theme.bg)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                }

                Button {
                    stopTimer()
                } label: {
                    Label("Stop", systemImage: "stop.fill")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(Theme.overtime)
                        .foregroundStyle(Theme.bg)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                }
            }
            .padding(.horizontal, 32)

        case .paused:
            HStack(spacing: 16) {
                Button {
                    resumeTimer()
                } label: {
                    Label("Resume", systemImage: "play.fill")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(Theme.accent)
                        .foregroundStyle(Theme.bg)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                }

                Button {
                    stopTimer()
                } label: {
                    Label("Stop", systemImage: "stop.fill")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(Theme.overtime)
                        .foregroundStyle(Theme.bg)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                }
            }
            .padding(.horizontal, 32)

        case .confirming:
            HStack(spacing: 16) {
                Button {
                    Task { await saveSession() }
                } label: {
                    Label("Save", systemImage: "checkmark")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(Theme.success)
                        .foregroundStyle(Theme.bg)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                }

                Button {
                    discardSession()
                } label: {
                    Label("Discard", systemImage: "xmark")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(Theme.border)
                        .foregroundStyle(Theme.fg)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                }
            }
            .padding(.horizontal, 32)
        }
    }

    // MARK: - Timer Logic

    private var progressFraction: CGFloat {
        let total = Double(plannedMinutes * 60)
        guard total > 0 else { return 0 }
        switch timerState {
        case .idle:
            return 1.0
        case .running, .paused:
            return CGFloat(Double(plannedMinutes * 60 - remainingSeconds) / total)
        case .overtime:
            // Pulse between 0.9 and 1.0 based on overtime seconds
            return 1.0
        case .confirming:
            return 1.0
        }
    }

    private var timeString: String {
        if timerState == .overtime {
            let mins = overtimeSeconds / 60
            let secs = overtimeSeconds % 60
            return String(format: "+%02d:%02d", mins, secs)
        } else {
            let display = max(remainingSeconds, 0)
            let mins = display / 60
            let secs = display % 60
            return String(format: "%02d:%02d", mins, secs)
        }
    }

    private func adjustTime(by minutes: Int) {
        plannedMinutes = max(5, min(120, plannedMinutes + minutes))
        remainingSeconds = plannedMinutes * 60
    }

    private func startTimer() {
        startTime = Date()
        remainingSeconds = plannedMinutes * 60
        elapsedSeconds = 0
        overtimeSeconds = 0
        timerState = .running
        startTicking()
    }

    private func pauseTimer() {
        timerState = .paused
        timerCancellable?.cancel()
    }

    private func resumeTimer() {
        timerState = timerState == .paused && remainingSeconds <= 0 ? .overtime : .running
        startTicking()
    }

    private func stopTimer() {
        timerCancellable?.cancel()
        timerState = .confirming
    }

    private func startTicking() {
        timerCancellable?.cancel()
        timerCancellable = Timer.publish(every: 1, on: .main, in: .common)
            .autoconnect()
            .sink { _ in
                tick()
            }
    }

    private func tick() {
        elapsedSeconds += 1
        if timerState == .overtime {
            overtimeSeconds += 1
        } else {
            remainingSeconds -= 1
            if remainingSeconds <= 0 {
                remainingSeconds = 0
                timerState = .overtime
            }
        }
    }

    private func saveSession() async {
        guard let habit = selectedHabit else { return }

        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]

        let session = NewSession(
            habitId: habit.id,
            plannedMinutes: plannedMinutes,
            actualSeconds: elapsedSeconds,
            overtimeSeconds: overtimeSeconds,
            completed: elapsedSeconds >= plannedMinutes * 60,
            startTime: formatter.string(from: startTime)
        )

        await dataService.createSession(session)
        resetTimer()
    }

    private func discardSession() {
        resetTimer()
    }

    private func resetTimer() {
        timerCancellable?.cancel()
        timerState = .idle
        remainingSeconds = plannedMinutes * 60
        elapsedSeconds = 0
        overtimeSeconds = 0
    }
}

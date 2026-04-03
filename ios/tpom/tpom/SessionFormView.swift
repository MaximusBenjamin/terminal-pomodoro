import SwiftUI

struct SessionFormView: View {
    var dataService: DataService
    var editing: SessionWithHabit?

    @Environment(\.dismiss) private var dismiss

    @State private var selectedHabitId: Int = 0
    @State private var sessionDate: Date = Date()
    @State private var startTime: Date = Date()
    @State private var durationHours: Int = 0
    @State private var durationMinutes: Int = 25
    @State private var isSaving = false

    private var isEditing: Bool { editing != nil }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    if dataService.habits.isEmpty {
                        Text("No habits available")
                            .foregroundStyle(Theme.muted)
                    } else {
                        Picker("Habit", selection: $selectedHabitId) {
                            ForEach(dataService.habits) { habit in
                                HStack(spacing: 8) {
                                    Circle()
                                        .fill(Color(hex: habit.color))
                                        .frame(width: 8, height: 8)
                                    Text(habit.name)
                                }
                                .tag(habit.id)
                            }
                        }
                        .pickerStyle(.menu)
                        .foregroundStyle(Theme.fg)
                        .tint(Theme.accent)
                    }
                } header: {
                    Text("Habit")
                        .foregroundStyle(Theme.muted)
                }
                .listRowBackground(Theme.border.opacity(0.2))

                Section {
                    DatePicker(
                        "Date",
                        selection: $sessionDate,
                        displayedComponents: .date
                    )
                    .foregroundStyle(Theme.fg)
                    .tint(Theme.accent)

                    DatePicker(
                        "Start Time",
                        selection: $startTime,
                        displayedComponents: .hourAndMinute
                    )
                    .foregroundStyle(Theme.fg)
                    .tint(Theme.accent)
                } header: {
                    Text("When")
                        .foregroundStyle(Theme.muted)
                }
                .listRowBackground(Theme.border.opacity(0.2))

                Section {
                    Stepper(
                        "Hours: \(durationHours)",
                        value: $durationHours,
                        in: 0...12
                    )
                    .foregroundStyle(Theme.fg)

                    Stepper(
                        "Minutes: \(durationMinutes)",
                        value: $durationMinutes,
                        in: 0...55,
                        step: 5
                    )
                    .foregroundStyle(Theme.fg)

                    HStack {
                        Text("Total")
                            .foregroundStyle(Theme.muted)
                        Spacer()
                        Text(formattedDuration)
                            .foregroundStyle(Theme.accent)
                            .fontWeight(.medium)
                    }
                } header: {
                    Text("Duration")
                        .foregroundStyle(Theme.muted)
                }
                .listRowBackground(Theme.border.opacity(0.2))
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bg)
            .navigationTitle(isEditing ? "Edit Session" : "Add Session")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                    .foregroundStyle(Theme.muted)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        save()
                    }
                    .fontWeight(.semibold)
                    .foregroundStyle(Theme.accent)
                    .disabled(isSaving || selectedHabitId == 0)
                }
            }
            .onAppear {
                prefill()
            }
        }
        .presentationBackground(Theme.bg)
    }

    // MARK: - Computed

    private var formattedDuration: String {
        let totalMinutes = durationHours * 60 + durationMinutes
        if durationHours > 0 {
            return "\(durationHours)h \(durationMinutes)m"
        } else {
            return "\(totalMinutes)m"
        }
    }

    // MARK: - Prefill

    private func prefill() {
        // Default to first habit if none selected
        if selectedHabitId == 0, let first = dataService.habits.first {
            selectedHabitId = first.id
        }

        guard let editing else { return }

        selectedHabitId = editing.session.habitId

        // Parse startTime ISO8601 string
        let parsed = parseISO8601(editing.session.startTime)
        sessionDate = parsed
        startTime = parsed

        // Calculate duration from actualSeconds
        let total = editing.session.actualSeconds
        durationHours = total / 3600
        durationMinutes = (total % 3600) / 60
        // Round to nearest 5
        let remainder = durationMinutes % 5
        if remainder >= 3 {
            durationMinutes = min(durationMinutes + (5 - remainder), 55)
        } else {
            durationMinutes = durationMinutes - remainder
        }
    }

    // MARK: - Save

    private func save() {
        isSaving = true

        // Combine sessionDate (date portion) with startTime (time portion)
        let calendar = Calendar.current
        let dateComponents = calendar.dateComponents([.year, .month, .day], from: sessionDate)
        let timeComponents = calendar.dateComponents([.hour, .minute], from: startTime)

        var combined = DateComponents()
        combined.year = dateComponents.year
        combined.month = dateComponents.month
        combined.day = dateComponents.day
        combined.hour = timeComponents.hour
        combined.minute = timeComponents.minute
        combined.second = 0

        let finalDate = calendar.date(from: combined) ?? Date()

        let actualSeconds = durationHours * 3600 + durationMinutes * 60
        let plannedMinutes = durationHours * 60 + durationMinutes

        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        let startTimeString = formatter.string(from: finalDate)

        let newSession = NewSession(
            habitId: selectedHabitId,
            plannedMinutes: plannedMinutes,
            actualSeconds: actualSeconds,
            overtimeSeconds: 0,
            completed: true,
            startTime: startTimeString
        )

        Task {
            if let editing {
                await dataService.updateSession(id: editing.session.id, session: newSession)
            } else {
                await dataService.createSession(newSession)
            }
            dismiss()
        }
    }

    // MARK: - Helpers

    private func parseISO8601(_ str: String) -> Date {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = f.date(from: str) { return d }
        f.formatOptions = [.withInternetDateTime]
        return f.date(from: str) ?? Date()
    }
}

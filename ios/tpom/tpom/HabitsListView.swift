import SwiftUI

struct HabitsListView: View {
    var dataService: DataService

    @State private var showAddSheet = false
    @State private var newHabitName = ""
    @State private var newHabitColor = "#7aa2f7"

    private let colorOptions = [
        "#7aa2f7", "#9ece6a", "#e0af68", "#f7768e",
        "#bb9af7", "#7dcfff", "#73daca", "#ff9e64"
    ]

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()

            VStack(spacing: 0) {
                // Header
                HStack {
                    Text("Habits")
                        .font(.title2.bold())
                        .foregroundStyle(Theme.accent)
                    Spacer()
                    Button {
                        showAddSheet = true
                    } label: {
                        Image(systemName: "plus.circle.fill")
                            .font(.title3)
                            .foregroundStyle(Theme.accent)
                    }
                }
                .padding(.horizontal)
                .padding(.top, 8)
                .padding(.bottom, 16)

                if dataService.habits.isEmpty {
                    Spacer()
                    VStack(spacing: 12) {
                        Image(systemName: "list.bullet")
                            .font(.system(size: 40))
                            .foregroundStyle(Theme.muted)
                        Text("No habits yet")
                            .foregroundStyle(Theme.muted)
                        Text("Tap + to add one")
                            .font(.caption)
                            .foregroundStyle(Theme.muted)
                    }
                    Spacer()
                } else {
                    List {
                        ForEach(dataService.habits) { habit in
                            HStack(spacing: 12) {
                                Circle()
                                    .fill(Color(hex: habit.color))
                                    .frame(width: 12, height: 12)

                                Text(habit.name)
                                    .foregroundStyle(Theme.fg)

                                Spacer()
                            }
                            .listRowBackground(Theme.border.opacity(0.15))
                        }
                        .onDelete { indexSet in
                            Task {
                                for index in indexSet {
                                    let habit = dataService.habits[index]
                                    await dataService.deleteHabit(id: habit.id)
                                }
                            }
                        }
                    }
                    .listStyle(.plain)
                    .scrollContentBackground(.hidden)
                }
            }
        }
        .sheet(isPresented: $showAddSheet) {
            addHabitSheet
        }
    }

    @ViewBuilder
    private var addHabitSheet: some View {
        NavigationStack {
            ZStack {
                Theme.bg.ignoresSafeArea()

                VStack(spacing: 24) {
                    TextField("Habit name", text: $newHabitName)
                        .textFieldStyle(.plain)
                        .padding(14)
                        .background(Theme.border.opacity(0.3))
                        .foregroundStyle(Theme.fg)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                        .padding(.horizontal)

                    // Color picker
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Color")
                            .font(.subheadline)
                            .foregroundStyle(Theme.muted)
                            .padding(.horizontal)

                        LazyVGrid(columns: Array(repeating: GridItem(.flexible()), count: 4), spacing: 12) {
                            ForEach(colorOptions, id: \.self) { color in
                                Circle()
                                    .fill(Color(hex: color))
                                    .frame(width: 40, height: 40)
                                    .overlay(
                                        Circle()
                                            .stroke(Theme.fg, lineWidth: color == newHabitColor ? 3 : 0)
                                    )
                                    .onTapGesture {
                                        newHabitColor = color
                                    }
                            }
                        }
                        .padding(.horizontal)
                    }

                    Spacer()
                }
                .padding(.top, 24)
            }
            .navigationTitle("New Habit")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        showAddSheet = false
                        newHabitName = ""
                    }
                    .foregroundStyle(Theme.muted)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Add") {
                        Task {
                            await dataService.addHabit(name: newHabitName, color: newHabitColor)
                            newHabitName = ""
                            newHabitColor = "#7aa2f7"
                            showAddSheet = false
                        }
                    }
                    .disabled(newHabitName.trimmingCharacters(in: .whitespaces).isEmpty)
                    .foregroundStyle(Theme.accent)
                }
            }
        }
        .presentationDetents([.medium])
    }
}

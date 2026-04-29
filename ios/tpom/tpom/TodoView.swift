import SwiftUI

struct TodoView: View {
    var dataService: DataService

    @State private var showAddSheet = false
    @State private var editingTodo: Todo?
    @State private var deleteCandidate: Todo?

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()

            VStack(spacing: 0) {
                header
                dateStrip
                Divider().background(Theme.border)
                listBody
            }

            // Floating add button (today only).
            if dataService.isTodoViewingToday {
                VStack {
                    Spacer()
                    HStack {
                        Spacer()
                        Button {
                            editingTodo = nil
                            showAddSheet = true
                        } label: {
                            Image(systemName: "plus.circle.fill")
                                .font(.system(size: 52))
                                .foregroundStyle(Theme.accent)
                                .background(Circle().fill(Theme.bg))
                        }
                        .padding(.trailing, 20)
                        .padding(.bottom, 24)
                    }
                }
            }
        }
        .sheet(isPresented: $showAddSheet) {
            TodoFormSheet(
                dataService: dataService,
                editing: editingTodo
            )
        }
        .confirmationDialog(
            "Delete this todo?",
            isPresented: Binding(
                get: { deleteCandidate != nil },
                set: { if !$0 { deleteCandidate = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button("Delete", role: .destructive) {
                if let todo = deleteCandidate {
                    Task { await dataService.deleteTodo(id: todo.id) }
                }
                deleteCandidate = nil
            }
            Button("Cancel", role: .cancel) {
                deleteCandidate = nil
            }
        } message: {
            Text(deleteCandidate?.text ?? "")
        }
    }

    // MARK: - Header

    private var header: some View {
        HStack {
            Text("Todo")
                .font(.title2.bold())
                .foregroundStyle(Theme.accent)
            Spacer()
        }
        .padding(.horizontal)
        .padding(.top, 8)
        .padding(.bottom, 8)
    }

    // MARK: - Date strip

    private var dateStrip: some View {
        HStack(spacing: 12) {
            Button {
                Task { await stepDay(by: -1) }
            } label: {
                Image(systemName: "chevron.left")
                    .foregroundStyle(Theme.accent)
            }

            Text(dateLabel)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(Theme.fg)
                .frame(minWidth: 120)

            Button {
                Task { await stepDay(by: 1) }
            } label: {
                Image(systemName: "chevron.right")
                    .foregroundStyle(canStepForward ? Theme.accent : Theme.muted)
            }
            .disabled(!canStepForward)

            Spacer()

            if !dataService.isTodoViewingToday {
                Button("Today") {
                    Task { await dataService.loadTodos(for: dataService.effectiveDay()) }
                }
                .font(.caption.weight(.medium))
                .foregroundStyle(Theme.accent)
            }
        }
        .padding(.horizontal)
        .padding(.bottom, 8)
    }

    private var dateLabel: String {
        if dataService.isTodoViewingToday {
            return "Today"
        }
        let f = DateFormatter()
        f.dateFormat = "EEE MMM d"
        return f.string(from: dataService.viewingTodoDate)
    }

    private var canStepForward: Bool {
        let today = dataService.effectiveDay()
        return dataService.viewingTodoDate < today
    }

    private func stepDay(by days: Int) async {
        let cal = Calendar.current
        guard let next = cal.date(byAdding: .day, value: days, to: dataService.viewingTodoDate) else { return }
        let today = dataService.effectiveDay()
        let target = next > today ? today : next
        await dataService.loadTodos(for: target)
    }

    // MARK: - List

    @ViewBuilder
    private var listBody: some View {
        if dataService.todos.isEmpty {
            Spacer()
            VStack(spacing: 12) {
                Image(systemName: "checklist")
                    .font(.system(size: 40))
                    .foregroundStyle(Theme.muted)
                Text(dataService.isTodoViewingToday
                     ? "No todos yet"
                     : "No todos this day")
                    .foregroundStyle(Theme.muted)
                if dataService.isTodoViewingToday {
                    Text("Tap + to add one")
                        .font(.caption)
                        .foregroundStyle(Theme.muted)
                }
            }
            Spacer()
        } else {
            List {
                ForEach(dataService.todos) { todo in
                    todoRow(todo)
                        .listRowBackground(Theme.border.opacity(0.15))
                        .contentShape(Rectangle())
                        .onTapGesture {
                            guard dataService.isTodoViewingToday else { return }
                            Task { await dataService.toggleTodo(id: todo.id, completed: !todo.completed) }
                        }
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if dataService.isTodoViewingToday {
                                Button(role: .destructive) {
                                    deleteCandidate = todo
                                } label: {
                                    Label("Delete", systemImage: "trash")
                                }
                                Button {
                                    editingTodo = todo
                                    showAddSheet = true
                                } label: {
                                    Label("Edit", systemImage: "pencil")
                                }
                                .tint(Theme.accent)
                            }
                        }
                }
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)
        }
    }

    @ViewBuilder
    private func todoRow(_ todo: Todo) -> some View {
        HStack(spacing: 12) {
            Image(systemName: todo.completed ? "checkmark.circle.fill" : "circle")
                .font(.title3)
                .foregroundStyle(todo.completed ? Theme.accent : Theme.muted)

            Text(todo.text)
                .strikethrough(todo.completed)
                .foregroundStyle(todo.completed ? Theme.muted : Theme.fg)

            Spacer()
        }
        .padding(.vertical, 4)
    }
}

// MARK: - Add/Edit sheet

private struct TodoFormSheet: View {
    var dataService: DataService
    var editing: Todo?

    @Environment(\.dismiss) private var dismiss
    @State private var text: String = ""
    @State private var isSaving = false

    private var isEditing: Bool { editing != nil }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("e.g. review chapter 3", text: $text, axis: .vertical)
                        .foregroundStyle(Theme.fg)
                        .lineLimit(1...4)
                } header: {
                    Text(isEditing ? "Edit todo" : "New todo")
                        .foregroundStyle(Theme.muted)
                }
                .listRowBackground(Theme.border.opacity(0.2))
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bg)
            .navigationTitle(isEditing ? "Edit Todo" : "Add Todo")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                        .foregroundStyle(Theme.muted)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") { save() }
                        .fontWeight(.semibold)
                        .foregroundStyle(Theme.accent)
                        .disabled(isSaving || trimmed.isEmpty)
                }
            }
            .onAppear {
                if let editing { text = editing.text }
            }
        }
        .presentationBackground(Theme.bg)
        .presentationDetents([.medium])
    }

    private var trimmed: String {
        text.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func save() {
        let value = trimmed
        guard !value.isEmpty else { return }
        isSaving = true
        Task {
            if let editing {
                await dataService.editTodo(id: editing.id, text: value)
            } else {
                await dataService.addTodo(text: value)
            }
            dismiss()
        }
    }
}

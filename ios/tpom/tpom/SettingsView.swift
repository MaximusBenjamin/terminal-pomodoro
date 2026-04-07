import SwiftUI

struct SettingsView: View {
    var dataService: DataService
    @State private var leeway = AppSettings.leewayDaysPerWeek

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()

            VStack(spacing: 0) {
                // Header
                HStack {
                    Text("Settings")
                        .font(.title2.bold())
                        .foregroundStyle(Theme.accent)
                    Spacer()
                }
                .padding(.horizontal)
                .padding(.top, 8)
                .padding(.bottom, 24)

                VStack(spacing: 20) {
                    // Leeway setting
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Streak Leeway")
                            .font(.headline)
                            .foregroundStyle(Theme.accent)

                        Text("Days per week you can miss without breaking your streak.")
                            .font(.caption)
                            .foregroundStyle(Theme.muted)

                        HStack(spacing: 20) {
                            Button {
                                if leeway > 0 {
                                    leeway -= 1
                                    Task { await dataService.saveLeeway(leeway) }
                                }
                            } label: {
                                Image(systemName: "minus.circle")
                                    .font(.title2)
                                    .foregroundStyle(leeway > 0 ? Theme.accent : Theme.muted)
                            }
                            .disabled(leeway == 0)

                            Text("\(leeway)")
                                .font(.system(size: 32, weight: .bold, design: .monospaced))
                                .foregroundStyle(Theme.accent)
                                .frame(minWidth: 40)

                            Button {
                                if leeway < 7 {
                                    leeway += 1
                                    Task { await dataService.saveLeeway(leeway) }
                                }
                            } label: {
                                Image(systemName: "plus.circle")
                                    .font(.title2)
                                    .foregroundStyle(leeway < 7 ? Theme.accent : Theme.muted)
                            }
                            .disabled(leeway == 7)

                            Spacer()

                            Text(leewayDescription)
                                .font(.caption)
                                .foregroundStyle(Theme.muted)
                                .multilineTextAlignment(.trailing)
                        }
                    }
                    .padding(16)
                    .background(Theme.border.opacity(0.3))
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                    .padding(.horizontal)
                }

                Spacer()
            }
        }
        .onAppear { leeway = AppSettings.leewayDaysPerWeek }
    }

    private var leewayDescription: String {
        if leeway == 0 { return "Must hit\nevery day" }
        if leeway == 7 { return "No streak\ntracking" }
        return "Target\n\(7 - leeway)/7 days"
    }
}

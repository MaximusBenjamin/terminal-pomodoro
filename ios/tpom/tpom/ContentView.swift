import SwiftUI

struct ContentView: View {
    @State private var dataService = DataService()
    @State private var selectedTab = 0
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        TabView(selection: $selectedTab) {
            TimerView(dataService: dataService)
                .tabItem { Label("Timer", systemImage: "timer") }
                .tag(0)

            StatsView(dataService: dataService)
                .tabItem { Label("Stats", systemImage: "chart.bar") }
                .tag(1)

            HabitsListView(dataService: dataService)
                .tabItem { Label("Habits", systemImage: "list.bullet") }
                .tag(2)

            LogView(dataService: dataService)
                .tabItem { Label("Log", systemImage: "clock.arrow.circlepath") }
                .tag(3)

            SettingsView()
                .tabItem { Label("Settings", systemImage: "gearshape") }
                .tag(4)
        }
        .tint(Theme.accent)
        .task {
            await dataService.fetchAll()
            await dataService.startRealtime()
        }
        .onChange(of: scenePhase) { _, newPhase in
            Task {
                if newPhase == .active {
                    await dataService.fetchAll()
                    await dataService.startRealtime()
                } else if newPhase == .background {
                    dataService.stopRealtime()
                }
            }
        }
    }
}

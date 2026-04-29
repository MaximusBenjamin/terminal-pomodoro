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

            TodoView(dataService: dataService)
                .tabItem { Label("Todo", systemImage: "checklist") }
                .tag(1)

            StatsView(dataService: dataService)
                .tabItem { Label("Stats", systemImage: "chart.bar") }
                .tag(2)

            HabitsListView(dataService: dataService)
                .tabItem { Label("Habits", systemImage: "list.bullet") }
                .tag(3)

            LogView(dataService: dataService)
                .tabItem { Label("Log", systemImage: "clock.arrow.circlepath") }
                .tag(4)

            SettingsView(dataService: dataService)
                .tabItem { Label("Settings", systemImage: "gearshape") }
                .tag(5)
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

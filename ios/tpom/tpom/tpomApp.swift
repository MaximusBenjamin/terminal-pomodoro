import SwiftUI
import Supabase

@main
struct tpomApp: App {
    @State private var isAuthenticated = false
    @State private var isLoading = true

    var body: some Scene {
        WindowGroup {
            Group {
                if isLoading {
                    // Blank dark screen while checking session — avoids flashing login screen
                    Theme.bg.ignoresSafeArea()
                } else if isAuthenticated {
                    ContentView()
                } else {
                    AuthView(isAuthenticated: $isAuthenticated)
                }
            }
            .preferredColorScheme(.dark)
            .task {
                // Single loop handles all auth events including initial session check
                for await (_, session) in supabase.auth.authStateChanges {
                    isAuthenticated = session != nil
                    isLoading = false
                    if let session = session {
                        WidgetDataStore.saveToken(WidgetAuthToken(
                            accessToken: session.accessToken,
                            refreshToken: session.refreshToken
                        ))
                    }
                }
            }
        }
    }
}

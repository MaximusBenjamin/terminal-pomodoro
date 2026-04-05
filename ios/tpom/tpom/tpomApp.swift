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
                // Listen for auth state changes — first event is always initialSession
                for await (event, session) in supabase.auth.authStateChanges {
                    isAuthenticated = session != nil
                    isLoading = false
                    if let session = session {
                        WidgetDataStore.saveToken(WidgetAuthToken(
                            accessToken: session.accessToken,
                            refreshToken: session.refreshToken
                        ))
                    }
                    // Only need the first event to determine initial state
                    if event == .initialSession { break }
                }
                // Continue listening for subsequent changes (sign in/out/refresh)
                for await (_, session) in supabase.auth.authStateChanges {
                    isAuthenticated = session != nil
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

import SwiftUI
import Supabase

@main
struct tpomApp: App {
    @State private var isAuthenticated = false

    var body: some Scene {
        WindowGroup {
            Group {
                if isAuthenticated {
                    ContentView()
                } else {
                    AuthView(isAuthenticated: $isAuthenticated)
                }
            }
            .preferredColorScheme(.dark)
            .task {
                // Check if already logged in
                if let session = try? await supabase.auth.session {
                    isAuthenticated = true
                    // Share token with widget extension
                    WidgetDataStore.saveToken(WidgetAuthToken(
                        accessToken: session.accessToken,
                        refreshToken: session.refreshToken
                    ))
                }
                // Listen for auth state changes
                for await (event, session) in supabase.auth.authStateChanges {
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

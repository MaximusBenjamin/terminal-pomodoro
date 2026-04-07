import Foundation
import Supabase

let supabase = SupabaseClient(
    supabaseURL: URL(string: "https://dqnvsgtksqhbrmqchlds.supabase.co")!,
    supabaseKey: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImRxbnZzZ3Rrc3FoYnJtcWNobGRzIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzUxNjQxNDMsImV4cCI6MjA5MDc0MDE0M30.ew7OMJlZQDdpi1d7Mfe6s7kcA6wuNtAIZxFGNdVBxjw",
    options: SupabaseClientOptions(
        auth: SupabaseClientOptions.AuthOptions(
            emitLocalSessionAsInitialSession: true
        )
    )
)

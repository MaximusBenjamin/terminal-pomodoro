import SwiftUI
import Supabase

struct AuthView: View {
    @Binding var isAuthenticated: Bool

    @State private var email = ""
    @State private var password = ""
    @State private var errorMessage: String?
    @State private var isLoading = false
    @State private var isSignUp = false

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()

            VStack(spacing: 32) {
                Spacer()

                // Logo
                VStack(spacing: 12) {
                    Image(systemName: "timer")
                        .font(.system(size: 56))
                        .foregroundStyle(Theme.accent)

                    Text("tpom")
                        .font(.largeTitle.bold())
                        .foregroundStyle(Theme.fg)

                    Text("Pomodoro Timer")
                        .font(.subheadline)
                        .foregroundStyle(Theme.muted)
                }

                // Form
                VStack(spacing: 16) {
                    TextField("Email", text: $email)
                        .textFieldStyle(.plain)
                        .keyboardType(.emailAddress)
                        .textContentType(.emailAddress)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .padding(14)
                        .background(Theme.border.opacity(0.3))
                        .foregroundStyle(Theme.fg)
                        .clipShape(RoundedRectangle(cornerRadius: 10))

                    SecureField("Password", text: $password)
                        .textFieldStyle(.plain)
                        .textContentType(isSignUp ? .newPassword : .password)
                        .padding(14)
                        .background(Theme.border.opacity(0.3))
                        .foregroundStyle(Theme.fg)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .padding(.horizontal, 32)

                // Error
                if let errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(Theme.overtime)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 32)
                }

                // Buttons
                VStack(spacing: 12) {
                    Button {
                        Task { await authenticate() }
                    } label: {
                        HStack {
                            if isLoading {
                                ProgressView()
                                    .tint(Theme.bg)
                            }
                            Text(isSignUp ? "Create Account" : "Sign In")
                                .fontWeight(.semibold)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(Theme.accent)
                        .foregroundStyle(Theme.bg)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                    }
                    .disabled(isLoading || email.isEmpty || password.isEmpty)
                    .padding(.horizontal, 32)

                    Button {
                        isSignUp.toggle()
                        errorMessage = nil
                    } label: {
                        Text(isSignUp ? "Already have an account? Sign In" : "Don't have an account? Sign Up")
                            .font(.footnote)
                            .foregroundStyle(Theme.muted)
                    }
                }

                Spacer()
                Spacer()
            }
        }
    }

    private func authenticate() async {
        isLoading = true
        errorMessage = nil

        do {
            if isSignUp {
                let response = try await supabase.auth.signUp(
                    email: email,
                    password: password
                )
                switch response {
                case .session:
                    isAuthenticated = true
                case .user:
                    errorMessage = "Check your email to confirm your account."
                }
            } else {
                _ = try await supabase.auth.signIn(
                    email: email,
                    password: password
                )
                isAuthenticated = true
            }
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }
}

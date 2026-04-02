package app

// Global key bindings (handled in app.go):
//   tab / shift+tab : switch between tabs (Timer, Stats, Habits)
//   q / ctrl+c      : quit the application
//
// View-specific bindings are handled by each sub-model:
//   Timer  : space (start/stop), r (reset), +/- (adjust duration)
//   Stats  : left/right (navigate weeks), h (toggle heatmap)
//   Habits : j/k (move cursor), enter (select), a (add), d (archive)

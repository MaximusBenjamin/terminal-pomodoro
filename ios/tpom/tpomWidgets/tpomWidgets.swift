import WidgetKit
import SwiftUI

@main
struct TpomWidgetBundle: WidgetBundle {
    var body: some Widget {
        SummaryWidget()
        HeatmapWidget()
        WeeklyChartWidget()
        WeeklyHabitWidget()
    }
}

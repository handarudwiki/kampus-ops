import type { App } from 'vue'
import KAlert from './components/KAlert.vue'
import KAvatar from './components/KAvatar.vue'
import KBadge from './components/KBadge.vue'
import KButton from './components/KButton.vue'
import KCard from './components/KCard.vue'
import KDot from './components/KDot.vue'
import KEmpty from './components/KEmpty.vue'
import KField from './components/KField.vue'
import KIconButton from './components/KIconButton.vue'
import KInput from './components/KInput.vue'
import KKeyValue from './components/KKeyValue.vue'
import KMenu from './components/KMenu.vue'
import KModal from './components/KModal.vue'
import KPageHeader from './components/KPageHeader.vue'
import KPagination from './components/KPagination.vue'
import KProgress from './components/KProgress.vue'
import KSearch from './components/KSearch.vue'
import KSegmented from './components/KSegmented.vue'
import KSelect from './components/KSelect.vue'
import KSkeleton from './components/KSkeleton.vue'
import KStatCard from './components/KStatCard.vue'
import KStatus from './components/KStatus.vue'
import KStepper from './components/KStepper.vue'
import KTable from './components/KTable.vue'
import KTextarea from './components/KTextarea.vue'
import KToolbar from './components/KToolbar.vue'
import KBarChart from './charts/KBarChart.vue'
import KGauge from './charts/KGauge.vue'
import KLegend from './charts/KLegend.vue'
import KLineChart from './charts/KLineChart.vue'
import KRing from './charts/KRing.vue'

export const components = {
  KAlert, KAvatar, KBadge, KButton, KCard, KDot, KEmpty, KField, KIconButton, KInput, KKeyValue, KMenu,
  KModal, KPageHeader, KPagination, KProgress, KSearch, KSegmented, KSelect, KSkeleton, KStatCard,
  KStatus, KStepper, KTable, KTextarea, KToolbar, KBarChart, KGauge, KLegend, KLineChart, KRing,
}

/** Registers every K* component globally so views never hand-roll a button or badge. */
export const DesignSystem = {
  install(app: App) {
    for (const [name, c] of Object.entries(components)) app.component(name, c)
  },
}

declare module 'vue' {
  export interface GlobalComponents {
    KAlert: typeof KAlert; KAvatar: typeof KAvatar; KBadge: typeof KBadge; KButton: typeof KButton
    KCard: typeof KCard; KDot: typeof KDot; KEmpty: typeof KEmpty; KField: typeof KField
    KIconButton: typeof KIconButton; KInput: typeof KInput; KKeyValue: typeof KKeyValue; KMenu: typeof KMenu
    KModal: typeof KModal; KPageHeader: typeof KPageHeader; KPagination: typeof KPagination
    KProgress: typeof KProgress; KSearch: typeof KSearch; KSegmented: typeof KSegmented; KSelect: typeof KSelect
    KSkeleton: typeof KSkeleton; KStatCard: typeof KStatCard; KStatus: typeof KStatus; KStepper: typeof KStepper
    KTable: typeof KTable; KTextarea: typeof KTextarea; KToolbar: typeof KToolbar; KBarChart: typeof KBarChart
    KGauge: typeof KGauge; KLegend: typeof KLegend; KLineChart: typeof KLineChart; KRing: typeof KRing
  }
}

export * from './format'
export * from './status'
export * from './feedback'
export { toneClass, SERIES_TONES, toneFor } from './tone'

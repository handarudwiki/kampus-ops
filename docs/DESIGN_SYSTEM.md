# KampusOps Design System

Built from the reference dashboard (`rekomendasi_dashboard_ui.png`): a light gray canvas, white rounded cards, small colored dots as category markers, big numbers with muted tails, black pill buttons, and tinted suggestion panels.

Live showcase: **`/design-system`** in the running app (public route).

## Where things live

| Concern | File | Rule |
|---|---|---|
| Tokens (color, radius, shadow, motion; light + dark) | `frontend/src/styles/tokens.css` | The only place raw color values exist |
| Tailwind mapping | `frontend/tailwind.config.js` | Palette is **replaced**, not extended — `bg-blue-500` does not exist |
| Tone → class map | `frontend/src/design-system/tone.ts` | `toneClass[tone].dot / soft / tint / text / bar / stroke` |
| Domain statuses | `frontend/src/design-system/status.ts` | Every status label + tone. Render with `<KStatus domain value>` only |
| API error messages | `status.ts` → `ERROR_TEXT` | Every business error code → one Indonesian sentence |
| Formatting | `frontend/src/design-system/format.ts` | `rupiah`, `rupiahShort`, `date`, `slot`, `bobot`, `ip`, `num` |
| Components | `frontend/src/design-system/components`, `charts` | Registered globally as `K*` |
| Loading / mutations | `frontend/src/composables/useAsync.ts` | `useAsync` for loads, `useAction` for mutations (toasts built in) |
| Toast / confirm | `frontend/src/design-system/feedback.ts` | No `alert()` / `confirm()` |

## Tokens

**Surfaces:** `canvas` (app background) → `surface` (cards) → `surface-2` (inputs, insets, hover) → `surface-3` (segmented track, active nav). `line` for hairlines.
**Ink:** `ink` (primary text and primary button), `ink-2` (secondary), `ink-3` (captions, muted number tails), `ink-on` (text on ink).
**Accent:** `accent` for links, focus rings, active nav icon, chart series 1.
**Tones:** `blue green amber red violet pink gray`, each with `DEFAULT` (dot/fill), `soft` (tinted bg) and `ink` (text on soft).

Semantics: green = done/approved/paid, amber = waiting/partial/warning, red = rejected/unpaid/error, blue = informational/finished, violet & pink = categories (PMB stages, bill types), gray = draft/neutral.

**Radius:** cards `rounded-card` (20px), controls `rounded-control` (12px), buttons/chips `rounded-pill`.
**Type:** Plus Jakarta Sans. Scale: `xs 12 · sm 13 · base 14 · md 16 · lg 20 · xl 28 · 2xl 40`. Card titles `text-md font-bold`; page titles `text-xl font-bold`; column headers `.ko-eyebrow`.
**Numbers:** always `.tabular`; IPK as `3,42`; money via `rupiah()`.
**Dark mode:** follows the OS by default; the sidebar toggle sets `data-theme`. Components need no `dark:` classes because tokens swap.

## Components

| Component | Use for |
|---|---|
| `KPageHeader` | First element of every page: title, one-line description, actions |
| `KCard` | Every content block. `to` adds the ↗ link; `flush` for tables |
| `KStatCard` | KPIs. Dot + label, big figure (muted tail), hint chip with trend |
| `KToolbar` + `KTable` (+ `KPagination`) | Every list. Custom cells via `#cell-<key>` |
| `KStatus` / `KBadge` | Status (`pill` in dense UI, `dot` in tables like the reference) |
| `KButton` / `KIconButton` | `primary` (ink) once per area; `secondary`, `soft`, `ghost`, `danger` |
| `KField` + `KInput` / `KSelect` / `KTextarea` / `KSearch` | All forms. `KField` owns label, hint and error |
| `KSegmented` | 2–5 views or filters (status tabs) |
| `KModal` | Create/edit forms. Becomes a bottom sheet on mobile |
| `KMenu` | Row actions, role switcher |
| `KAlert` | Inline, contextual messages (tinted panels) |
| `KStepper` | Lifecycles: PMB, KRS |
| `KKeyValue` | Detail panels |
| `KEmpty` / `KSkeleton` | Empty and loading states |
| `KBarChart` / `KLineChart` / `KGauge` / `KRing` / `KLegend` | Charts; series colors follow `SERIES_TONES` |

## Page pattern

```vue
<KPageHeader title="…" description="…"><template #actions><KButton>Tambah</KButton></template></KPageHeader>
<KCard flush>
  <KToolbar><KSearch v-model="q" /><KSegmented v-model="status" :options="…" /></KToolbar>
  <KTable :columns :rows :loading>
    <template #cell-status="{ value }"><KStatus domain="krs" :value="value" variant="dot" /></template>
  </KTable>
</KCard>
<KModal v-model="open" title="…">…<template #footer>Batal · Simpan</template></KModal>
```

Loading shows a skeleton, failures show a toast (`useAsync`), mutations go through `useAction().run(fn, 'Pesan sukses')`, and destructive or reasoned actions go through `confirm({ withReason: true })`.

## Responsive

Mobile first down to **360px**. The sidebar becomes a drawer under `lg`, tables scroll horizontally and drop `hideMobile` columns, modals become bottom sheets, and page actions wrap under the title.

## Adding something new

1. New status? Add it to `STATUS` in `status.ts`, never inline a color.
2. New error code? Add it to `ERROR_TEXT`.
3. Need a color that isn't a token? Don't add one. Pick the nearest semantic tone.
4. New component? Put it in `design-system/components`, register it in `design-system/index.ts`, and show it on `/design-system`.

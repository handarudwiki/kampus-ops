/** @type {import('tailwindcss').Config} */
const c = (name) => `rgb(var(--ko-${name}) / <alpha-value>)`
const tones = ['blue', 'green', 'amber', 'red', 'violet', 'pink', 'gray']

export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  // No `dark:` variants needed: tokens.css swaps every token for dark mode.
  theme: {
    // Restrict the palette to tokens so nobody can reach for bg-blue-500.
    colors: {
      transparent: 'transparent',
      current: 'currentColor',
      canvas: c('canvas'),
      surface: { DEFAULT: c('surface'), 2: c('surface-2'), 3: c('surface-3') },
      line: { DEFAULT: c('line'), strong: c('line-strong') },
      ink: { DEFAULT: c('ink'), 2: c('ink-2'), 3: c('ink-3'), on: c('on-ink') },
      accent: { DEFAULT: c('accent'), soft: c('accent-soft') },
      ...Object.fromEntries(
        tones.map((t) => [t, { DEFAULT: c(t), soft: c(`${t}-soft`), ink: c(`${t}-ink`) }]),
      ),
    },
    fontFamily: {
      sans: ['"Plus Jakarta Sans"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
      mono: ['"JetBrains Mono"', 'ui-monospace', 'monospace'],
    },
    // Type scale: caption 12 · body 14 · title 16 · heading 20 · display 28/40
    fontSize: {
      xs: ['12px', { lineHeight: '16px' }],
      sm: ['13px', { lineHeight: '18px' }],
      base: ['14px', { lineHeight: '20px' }],
      md: ['16px', { lineHeight: '22px' }],
      lg: ['20px', { lineHeight: '28px' }],
      xl: ['28px', { lineHeight: '34px' }],
      '2xl': ['40px', { lineHeight: '44px' }],
    },
    extend: {
      borderRadius: {
        card: 'var(--ko-radius-card)',
        control: 'var(--ko-radius-control)',
        pill: 'var(--ko-radius-pill)',
      },
      boxShadow: { card: 'var(--ko-shadow-card)', pop: 'var(--ko-shadow-pop)' },
      transitionTimingFunction: { ko: 'var(--ko-ease)' },
      transitionDuration: { ko: 'var(--ko-dur)' },
    },
  },
  plugins: [],
}

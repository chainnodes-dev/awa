/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  theme: {
    extend: {
      colors: {
        surface: {
          0: 'var(--color-bg)',
          1: 'var(--color-surface)',
          2: 'var(--color-surface-2)',
        },
        border: 'var(--color-border)',
        accent: {
          DEFAULT: 'var(--color-accent)',
          hover: 'var(--color-accent)',
        },
        text: {
          DEFAULT: 'var(--color-text)',
          muted: 'var(--color-text-muted)',
        },
        state: {
          initial: '#22c55e',
          intermediate: '#6366f1',
          hitl: '#f59e0b',
          terminal: '#64748b',
          failed: '#ef4444',
          active: '#a78bfa',
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
    }
  },
  plugins: []
}

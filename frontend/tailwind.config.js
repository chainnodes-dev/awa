/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  theme: {
    extend: {
      colors: {
        surface: {
          0: '#0c0e14',
          1: '#12151e',
          2: '#1a1d27',
          3: '#21253200',
        },
        border: '#2a2d3a',
        accent: {
          DEFAULT: '#6366f1',
          hover: '#818cf8',
          muted: '#6366f120',
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
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'glow': 'glow 2s ease-in-out infinite alternate',
      },
      keyframes: {
        glow: {
          from: { boxShadow: '0 0 8px 2px rgba(99, 102, 241, 0.4)' },
          to:   { boxShadow: '0 0 20px 6px rgba(99, 102, 241, 0.8)' },
        }
      }
    }
  },
  plugins: []
}

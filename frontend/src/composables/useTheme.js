import { ref, watchEffect } from 'vue'

export function useTheme() {
  const theme = ref(localStorage.getItem('chainnodes-theme') || localStorage.getItem('phaxa-theme') || 'system')

  function setTheme(val) {
    theme.value = val
    localStorage.setItem('chainnodes-theme', val)
  }

  watchEffect(() => {
    const isDark = 
      theme.value === 'dark' || 
      (theme.value === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)

    if (isDark) {
      document.documentElement.classList.remove('light')
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
      document.documentElement.classList.add('light')
    }
  })

  // Listen for system changes if in system mode
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (theme.value === 'system') {
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      if (isDark) {
        document.documentElement.classList.add('dark')
        document.documentElement.classList.remove('light')
      } else {
        document.documentElement.classList.add('light')
        document.documentElement.classList.remove('dark')
      }
    }
  })

  return { theme, setTheme }
}

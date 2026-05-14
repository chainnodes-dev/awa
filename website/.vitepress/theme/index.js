import DefaultTheme from 'vitepress/theme'
import './custom.css'
import HomeHero from './components/HomeHero.vue'
import ContactForm from './components/ContactForm.vue'

export default {
  ...DefaultTheme,
  enhanceApp({ app }) {
    app.component('HomeHero', HomeHero)
    app.component('ContactForm', ContactForm)
  }
}

import { initializeApp } from 'firebase/app'
import { getFirestore } from 'firebase/firestore'

// Your Firebase web app configuration.
// These credentials can be pasted here or loaded via VITE_ environment variables.
const firebaseConfig = {
  apiKey: "AIzaSyD4MLJfXpvt7s1OOdpH8OzdUIgeoQWyYeI",
  authDomain: "fb-agentic-workflow-automation.firebaseapp.com",
  projectId: "fb-agentic-workflow-automation",
  storageBucket: "fb-agentic-workflow-automation.firebasestorage.app",
  messagingSenderId: "1011930557878",
  appId: "1:1011930557878:web:b12c7c764de933892c2517",
  measurementId: "G-C6684PGY7N"
};

let db = null

// Guard: Only initialize Firebase on the client (browser) side
// VitePress runs in Node.js during the build phase, where window/Firebase auth won't work
if (typeof window !== 'undefined') {
  try {
    const app = initializeApp(firebaseConfig)
    db = getFirestore(app)
  } catch (error) {
    console.error('Firebase initialization failed:', error)
  }
}

export { db }

import { initializeApp } from 'firebase/app'
import { getFirestore } from 'firebase/firestore'

// Your Firebase web app configuration.
// These credentials can be pasted here or loaded via VITE_ environment variables.
const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY || "YOUR_API_KEY",
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN || "fb-agentic-workflow-automation.firebaseapp.com",
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID || "fb-agentic-workflow-automation",
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET || "fb-agentic-workflow-automation.firebasestorage.app",
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID || "YOUR_MESSAGING_SENDER_ID",
  appId: import.meta.env.VITE_FIREBASE_APP_ID || "YOUR_APP_ID"
}

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

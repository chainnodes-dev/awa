<script setup>
import { ref } from 'vue'
import { db } from '../firebase'
import { collection, addDoc } from 'firebase/firestore'

const name = ref('')
const email = ref('')
const message = ref('')

const isSubmitting = ref(false)
const isSubmitted = ref(false)
const errorMessage = ref('')

const validateEmail = (emailVal) => {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailVal)
}

const handleSubmit = async () => {
  errorMessage.value = ''
  
  if (!name.value.trim()) {
    errorMessage.value = 'Please enter your name.'
    return
  }
  if (!validateEmail(email.value)) {
    errorMessage.value = 'Please enter a valid email address.'
    return
  }
  if (!message.value.trim()) {
    errorMessage.value = 'Please tell us about your use case.'
    return
  }

  isSubmitting.value = true

  try {
    // Check if Firebase is fully initialized with custom credentials
    const isFirebaseConfigured = db && db.app.options.apiKey && db.app.options.apiKey !== 'YOUR_API_KEY'
    
    if (isFirebaseConfigured) {
      await addDoc(collection(db, 'contacts'), {
        name: name.value,
        email: email.value,
        message: message.value,
        submittedAt: new Date().toISOString()
      })
    } else {
      console.warn('Firebase Firestore is not configured. Simulating submission.')
      await new Promise(resolve => setTimeout(resolve, 1000))
    }
    
    isSubmitted.value = true
  } catch (err) {
    console.error('Firestore submission failed:', err)
    errorMessage.value = 'Something went wrong. Please try again.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <section class="contact-section">
    <div class="contact-card premium-card">
      <Transition name="fade-slide" mode="out-in">
        <div v-if="!isSubmitted" key="form-view">
          <h2 class="title">Let's Build the <span class="gradient">Future</span></h2>
          <form class="contact-form" @submit.prevent="handleSubmit">
            <div class="input-group">
              <input 
                type="text" 
                placeholder="Full Name" 
                class="input" 
                v-model="name"
                :disabled="isSubmitting"
              />
            </div>
            <div class="input-group">
              <input 
                type="email" 
                placeholder="Business Email" 
                class="input" 
                v-model="email"
                :disabled="isSubmitting"
              />
            </div>
            <div class="input-group">
              <textarea 
                placeholder="Tell us about your use case" 
                class="input textarea" 
                rows="4"
                v-model="message"
                :disabled="isSubmitting"
              ></textarea>
            </div>
            
            <p v-if="errorMessage" class="error-text">{{ errorMessage }}</p>

            <button type="submit" class="submit-btn" :disabled="isSubmitting">
              <span v-if="isSubmitting" class="loader"></span>
              <span v-else>Schedule a Demo</span>
            </button>
          </form>
        </div>

        <div v-else key="success-view" class="success-content">
          <div class="success-icon-wrapper">
            <svg class="checkmark" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 52 52">
              <circle class="checkmark__circle" cx="26" cy="26" r="25" fill="none"/>
              <path class="checkmark__check" fill="none" d="M14.1 27.2l7.1 7.2 16.7-16.8"/>
            </svg>
          </div>
          <h2 class="success-title">Request <span class="gradient">Received!</span></h2>
          <p class="success-text">
            Thank you, <strong>{{ name }}</strong>. We have received your request and will reach out to <strong>{{ email }}</strong> within 24 hours to schedule your demo.
          </p>
        </div>
      </Transition>
    </div>
  </section>
</template>

<style scoped>
.contact-section {
  padding: 80px 24px;
  display: flex;
  justify-content: center;
}

.contact-card {
  width: 100%;
  max-width: 800px;
  padding: 48px;
  text-align: center;
  position: relative;
  overflow: hidden;
}

@media (min-width: 640px) {
  .contact-card { padding: 64px; }
}

.title {
  font-family: 'Outfit', sans-serif;
  font-size: 32px;
  font-weight: 800;
  margin-bottom: 40px;
}

@media (min-width: 640px) {
  .title { font-size: 48px; }
}

.gradient {
  background: linear-gradient(135deg, #a855f7 0%, #ec4899 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.contact-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 540px;
  margin: 0 auto;
}

.input {
  width: 100%;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  padding: 14px 20px;
  color: var(--vp-c-text-1);
  font-size: 16px;
  transition: all 0.2s ease;
}

.input:focus {
  outline: none;
  border-color: var(--vp-c-brand);
  background: var(--vp-c-bg-soft);
}

.textarea {
  resize: none;
}

.error-text {
  color: #f87171;
  font-size: 14px;
  margin: 0;
  text-align: left;
  max-width: 540px;
  margin: 0 auto;
}

.submit-btn {
  background: var(--vp-c-brand);
  color: #fff;
  padding: 16px;
  border-radius: 12px;
  font-weight: 700;
  font-size: 16px;
  margin-top: 8px;
  transition: all 0.2s ease;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.submit-btn:hover:not(:disabled) {
  background: var(--vp-c-brand-dark);
  transform: translateY(-2px);
  box-shadow: 0 10px 20px rgba(168, 85, 247, 0.2);
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

/* Success View Styling */
.success-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px 0;
}

.success-title {
  font-family: 'Outfit', sans-serif;
  font-size: 32px;
  font-weight: 800;
  margin: 24px 0 16px;
}

@media (min-width: 640px) {
  .success-title { font-size: 40px; }
}

.success-text {
  font-size: 18px;
  line-height: 1.6;
  color: var(--vp-c-text-2);
  max-width: 500px;
  margin: 0 auto;
}

/* Checkmark Animation */
.success-icon-wrapper {
  width: 80px;
  height: 80px;
}

.checkmark__circle {
  stroke-dasharray: 166;
  stroke-dashoffset: 166;
  stroke-width: 2;
  stroke-miterlimit: 10;
  stroke: #a855f7;
  fill: none;
  animation: stroke 0.6s cubic-bezier(0.65, 0, 0.45, 1) forwards;
}

.checkmark {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  display: block;
  stroke-width: 2;
  stroke: #fff;
  stroke-miterlimit: 10;
  box-shadow: inset 0px 0px 0px #a855f7;
  animation: fill .4s ease-in-out .4s forwards, scale .3s ease-in-out .9s forwards;
}

.checkmark__check {
  transform-origin: 50% 50%;
  stroke-dasharray: 48;
  stroke-dashoffset: 48;
  animation: stroke 0.3s cubic-bezier(0.65, 0, 0.45, 1) 0.8s forwards;
}

@keyframes stroke {
  100% { stroke-dashoffset: 0; }
}
@keyframes scale {
  0%, 100% { transform: none; }
  50% { transform: scale3d(1.1, 1.1, 1); }
}
@keyframes fill {
  100% { box-shadow: inset 0px 0px 0px 40px #a855f7; }
}

/* Transition Animations */
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.3s ease;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-20px);
}

/* Spinner Loader */
.loader {
  width: 20px;
  height: 20px;
  border: 3px solid rgba(255, 255, 255, 0.3);
  border-radius: 50%;
  border-top-color: #fff;
  animation: spin 1s ease-in-out infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>


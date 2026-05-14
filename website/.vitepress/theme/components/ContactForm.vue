<template>
  <section class="contact-section">
    <div class="contact-card premium-card">
      <h2 class="title">Let's Build the <span class="gradient">Future</span></h2>
      <form class="contact-form" @submit.prevent>
        <div class="input-group">
          <input type="text" placeholder="Full Name" class="input" />
        </div>
        <div class="input-group">
          <input type="email" placeholder="Business Email" class="input" />
        </div>
        <div class="input-group">
          <textarea placeholder="Tell us about your use case" class="input textarea" rows="4"></textarea>
        </div>
        <button type="submit" class="submit-btn">
          Schedule a Demo
        </button>
      </form>
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
}

.submit-btn:hover {
  background: var(--vp-c-brand-dark);
  transform: translateY(-2px);
  box-shadow: 0 10px 20px rgba(168, 85, 247, 0.2);
}
</style>

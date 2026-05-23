<script setup>
import { withBase } from 'vitepress'
</script>

<template>
  <div class="home-hero">
    <div class="container">
      <div class="content">
        
        <h1 class="name">
          The Future of 
          <span class="gradient">Autonomous</span>
          Logic.
        </h1>
        
        <p class="text">
          Chain Nodes is the industrial-grade autonomous agent platform. 
          Orchestrate neural workflows, bridge data silos with MCP, 
          and scale with Temporal precision.
        </p>

        <div class="actions">
          <a href="/guide/getting-started" class="action primary">
            Get Started
          </a>
        </div>
      </div>

      <div class="image-container">
        <div class="glow"></div>
        <div class="orb-container animate-float">
          <img 
            :src="withBase('/assets/logos/logo-card.svg')" 
            alt="Chain Nodes Logo" 
            class="hero-image" 
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home-hero {
  padding: 80px 24px;
  overflow: hidden;
  position: relative;
}

.container {
  max-width: 1152px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

@media (min-width: 960px) {
  .container {
    flex-direction: row;
    text-align: left;
    justify-content: space-between;
  }
}

.content {
  flex: 1;
  z-index: 10;
}



.name {
  font-size: 48px;
  font-weight: 800;
  line-height: 1.1;
  margin-bottom: 24px;
  letter-spacing: -0.02em;
}

@media (min-width: 640px) {
  .name { font-size: 64px; }
}

@media (min-width: 960px) {
  .name { font-size: 80px; }
}

.gradient {
  background: linear-gradient(135deg, #a855f7 0%, #ec4899 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  display: block;
}

.text {
  font-size: 18px;
  color: var(--vp-c-text-2);
  max-width: 540px;
  line-height: 1.6;
  margin-bottom: 40px;
}

@media (min-width: 960px) {
  .text { font-size: 20px; }
}

.actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

@media (min-width: 960px) {
  .actions { justify-content: flex-start; }
}

.action {
  padding: 12px 32px;
  border-radius: 12px;
  font-weight: 700;
  transition: all 0.2s ease;
}

.action.primary {
  background: var(--vp-c-brand);
  color: white;
}

.action.primary:hover {
  background: var(--vp-c-brand-dark);
  transform: translateY(-2px);
}

.action.secondary {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: var(--vp-c-text-1);
}

.action.secondary:hover {
  background: rgba(255, 255, 255, 0.1);
}

.image-container {
  position: relative;
  margin-top: 64px;
  width: 280px;
  height: 280px;
}

@media (min-width: 960px) {
  .image-container {
    margin-top: 0;
    width: 420px;
    height: 420px;
  }
}

.hero-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
  z-index: 2;
  position: relative;
}

.orb-container {
  position: relative;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.glow {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 120%;
  height: 120%;
  background: radial-gradient(circle, rgba(168, 85, 247, 0.2) 0%, transparent 70%);
  filter: blur(40px);
  z-index: 1;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-20px); }
}

@keyframes fade-in {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.animate-fade-in {
  animation: fade-in 0.8s ease-out forwards;
}
</style>

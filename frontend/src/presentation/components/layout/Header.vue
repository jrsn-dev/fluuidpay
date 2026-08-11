<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const isScrolled = ref(false)
const mobileMenuOpen = ref(false)

const handleScroll = () => {
  isScrolled.value = window.scrollY > 20
}

onMounted(() => {
  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<template>
  <header 
    class="fixed top-0 left-0 right-0 z-50 transition-all duration-300"
    :class="isScrolled ? 'bg-bg-hero/80 backdrop-blur-md border-b border-white/5' : 'bg-transparent'"
  >
    <div class="container mx-auto px-6 h-[70px] flex items-center justify-between">
      <!-- Logo -->
      <a href="/" class="flex items-center gap-3 text-xl font-bold text-white">
        <div class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="#0a0f1c" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <rect x="1" y="4" width="22" height="16" rx="2" ry="2"></rect>
            <line x1="1" y1="10" x2="23" y2="10"></line>
          </svg>
        </div>
        PayIntegra
      </a>

      <!-- Desktop Nav -->
      <nav class="hidden md:flex items-center gap-8">
        <a href="#inicio" class="text-sm font-medium text-text-light hover:text-white transition-colors">Início</a>
        <a href="#funcionalidades" class="text-sm font-medium text-text-light hover:text-white transition-colors">Funcionalidades</a>
        <a href="#integracoes" class="text-sm font-medium text-text-light hover:text-white transition-colors">Integrações</a>
        <a href="#precos" class="text-sm font-medium text-text-light hover:text-white transition-colors">Preços</a>
        <a href="#docs" class="text-sm font-medium text-text-light hover:text-white transition-colors">Documentação</a>
      </nav>

      <!-- Actions -->
      <div class="hidden md:flex items-center gap-3">
        <button class="btn-outline px-5 py-2 text-sm">Entrar</button>
        <button class="btn-primary px-5 py-2 text-sm">Começar Agora</button>
      </div>

      <!-- Mobile Toggle -->
      <button 
        @click="mobileMenuOpen = !mobileMenuOpen"
        class="md:hidden p-2 text-white"
      >
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path v-if="!mobileMenuOpen" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
          <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
        </svg>
      </button>
    </div>

    <!-- Mobile Menu -->
    <div v-if="mobileMenuOpen" class="md:hidden bg-bg-section border-t border-white/5">
      <nav class="container mx-auto px-6 py-4 flex flex-col gap-4">
        <a href="#inicio" class="text-text-light hover:text-white">Início</a>
        <a href="#funcionalidades" class="text-text-light hover:text-white">Funcionalidades</a>
        <a href="#integracoes" class="text-text-light hover:text-white">Integrações</a>
        <a href="#precos" class="text-text-light hover:text-white">Preços</a>
        <button class="btn-outline w-full justify-center">Entrar</button>
        <button class="btn-primary w-full justify-center">Começar Agora</button>
      </nav>
    </div>
  </header>
</template>

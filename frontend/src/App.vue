<script setup>
import { onMounted } from 'vue'
import Header from '@/presentation/components/layout/Header.vue'
import Hero from '@/presentation/components/sections/Hero.vue'
import Partners from '@/presentation/components/sections/Partners.vue'
import Services from '@/presentation/components/sections/Services.vue'
import HowItWorks from '@/presentation/components/sections/HowItWorks.vue'
import TechSection from '@/presentation/components/sections/TechSection.vue'
import Stats from '@/presentation/components/sections/Stats.vue'
import CTA from '@/presentation/components/sections/CTA.vue'
import Footer from '@/presentation/components/layout/Footer.vue'
import { initAnimations } from '@/presentation/composables/useAnimations'

onMounted(() => {
  initAnimations()
})
</script>

<template>
  <div class="min-h-screen bg-bg-hero">
    <Header />
    <main>
      <Hero />
      <Partners />
      <Services />
      <HowItWorks />
      <TechSection />
      <Stats />
      <CTA />
    </main>
    <Footer />
  </div>
</template>

<template>
  <section class="py-16 bg-bg-section">
    <div class="container mx-auto px-6">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-8 text-center">
        <div v-for="stat in stats" :key="stat.label" class="stat-item">
          <div class="stat-num text-4xl font-extrabold text-primary mb-1" :data-target="stat.value">{{ stat.value }}</div>
          <div class="text-sm text-text-light">{{ stat.label }}</div>
        </div>
      </div>
    </div>
  </section>
</template>
<script setup>
const stats = [
  { value: '+10K', label: 'Empresas integradas' },
  { value: '+5M', label: 'Transações processadas' },
  { value: '99.9%', label: 'Uptime garantido' },
  { value: '24/7', label: 'Suporte especializado' }
]
</script>

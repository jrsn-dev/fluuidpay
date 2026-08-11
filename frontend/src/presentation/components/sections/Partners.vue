<template>
  <section class="py-10 bg-bg-section border-y border-white/5">
    <div class="container mx-auto px-6 text-center">
      <p class="text-sm text-text-dim mb-7">Conecte-se com os principais gateways do mercado</p>
      <div class="flex flex-wrap justify-center items-center gap-12">
        <span v-for="partner in partners" :key="partner" class="text-lg font-bold text-slate-600 opacity-50 hover:opacity-100 transition-opacity cursor-default">{{ partner }}</span>
      </div>
    </div>
  </section>
</template>

<script setup>
const partners = ['stripe', 'pagar.me', 'mercado pago', 'asaas', 'PagSeguro', 'cielo', 'getnet', 'mollie']
</script>

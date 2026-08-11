<template>
  <section id="funcionalidades" class="py-24 bg-white">
    <div class="container mx-auto px-6">
      <div class="text-center max-w-2xl mx-auto mb-14">
        <h2 class="text-3xl font-bold text-bg-hero mb-3">Tudo que você precisa para gerenciar seu negócio</h2>
        <p class="text-text-dim">Uma plataforma completa para vender mais e simplificar sua operação</p>
      </div>
      
      <div class="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div v-for="service in services" :key="service.title" class="bg-white border border-slate-200 rounded-2xl p-8 hover:-translate-y-1 hover:shadow-[0_10px_25px_-5px_rgba(34,197,94,0.15)] hover:border-primary/30 transition-all duration-300 group">
          <div class="w-12 h-12 bg-green-100 rounded-xl flex items-center justify-center text-primary mb-5 group-hover:scale-110 transition-transform">
            <component :is="service.icon" class="w-6 h-6" />
          </div>
          <h3 class="text-lg font-bold text-bg-hero mb-2">{{ service.title }}</h3>
          <p class="text-sm text-text-dim leading-relaxed">{{ service.description }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
const services = [
  {
    title: 'Carrinho de Compras',
    description: 'Carrinho completo e personalizável com cupons, frete automático, cálculo de impostos e recuperação de vendas perdidas.',
    icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="9" cy="21" r="1"/><circle cx="20" cy="21" r="1"/><path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6"/></svg>'
  },
  {
    title: 'Gestão de Clientes',
    description: 'Cadastro completo com histórico de compras, segmentação inteligente e programa de fidelidade integrado.',
    icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>'
  },
  {
    title: 'Catálogo de Produtos',
    description: 'Organize seus produtos com variantes, categorias, estoque sincronizado e importação em lote via planilha.',
    icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>'
  },
  {
    title: 'Integração de Pagamentos',
    description: 'Múltiplos gateways via API unificada com split de pagamento, regras de routing e conciliação automática.',
    icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="1" y="4" width="22" height="16" rx="2" ry="2"/><line x1="1" y1="10" x2="23" y2="10"/></svg>'
  }
]
</script>

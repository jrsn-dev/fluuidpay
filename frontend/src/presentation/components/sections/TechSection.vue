<template>
  <section class="py-24 bg-bg-hero">
    <div class="container mx-auto px-6">
      <div class="grid lg:grid-cols-2 gap-8">
        <!-- API Panel -->
        <div class="bg-black border border-slate-800 rounded-xl overflow-hidden">
          <div class="p-6 pb-0">
            <h3 class="text-lg font-bold text-white mb-1">API Poderosa e Simples</h3>
            <p class="text-xs text-text-dim mb-4">Integração rápida com nossa API RESTful</p>
            <div class="inline-flex items-center gap-2 mb-4">
              <span class="px-2.5 py-1 bg-primary/10 border border-primary/20 rounded text-[10px] font-mono font-bold text-primary">POST</span>
              <span class="text-xs font-mono text-text-light">/api/v1/pagamentos</span>
              <span class="px-2.5 py-1 border border-primary rounded text-[9px] font-mono font-medium text-primary">200 OK</span>
            </div>
          </div>
          <div class="px-6 pb-4">
            <pre class="font-mono text-xs leading-relaxed text-text-light"><code><span class="text-slate-500">{</span>
  <span class="text-slate-400">"id"</span>: <span class="text-primary">"pay_9f8a7b6c"</span>,
  <span class="text-slate-400">"cliente_id"</span>: <span class="text-primary">"cli_123456"</span>,
  <span class="text-slate-400">"itens"</span>: <span class="text-slate-500">[</span>
    <span class="text-slate-500">{</span>
      <span class="text-slate-400">"nome"</span>: <span class="text-primary">"Fones Bluetooth"</span>,
      <span class="text-slate-400">"quantidade"</span>: <span class="text-amber-500">1</span>,
      <span class="text-slate-400">"valor"</span>: <span class="text-amber-500">189.90</span>
    <span class="text-slate-500">}</span>
  <span class="text-slate-500">]</span>,
  <span class="text-slate-400">"pagamento"</span>: <span class="text-slate-500">{</span>
    <span class="text-slate-400">"metodo"</span>: <span class="text-primary">"pix"</span>,
    <span class="text-slate-400">"status"</span>: <span class="text-primary">"aprovado"</span>,
    <span class="text-slate-400">"valor_total"</span>: <span class="text-amber-500">215.70</span>
  <span class="text-slate-500">}</span>
<span class="text-slate-500">}</span></code></pre>
          </div>
          <a href="#" class="inline-flex items-center gap-1.5 px-6 py-4 text-sm text-primary font-medium hover:gap-2.5 transition-all">Ver Documentação Completa <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg></a>
        </div>

        <!-- Checkout Panel -->
        <div class="bg-white rounded-xl overflow-hidden shadow-[0_25px_50px_rgba(0,0,0,0.3)]">
          <div class="p-6 pb-0">
            <h3 class="text-lg font-bold text-bg-hero mb-1">Checkout Personalizável</h3>
            <p class="text-xs text-text-dim mb-4">Experiência de compra otimizada para conversão</p>
          </div>
          <div class="px-6 pb-6">
            <div class="bg-slate-100 rounded-lg p-4 mb-4">
              <h4 class="text-sm font-semibold text-bg-hero mb-3">Seu Pedido</h4>
              <div class="flex justify-between text-xs text-slate-600 py-1.5"><span>Fones Bluetooth Pro</span><span class="font-medium text-bg-hero">R$ 189,90</span></div>
              <div class="flex justify-between text-xs text-slate-600 py-1.5"><span>Frete</span><span class="font-medium text-bg-hero">R$ 15,90</span></div>
              <hr class="border-dashed border-slate-300 my-2" />
              <div class="flex justify-between items-center"><span class="text-sm font-semibold text-bg-hero">Total</span><span class="text-lg font-extrabold text-bg-hero">R$ 205,80</span></div>
            </div>
            <div class="space-y-2">
              <div v-for="(pm, i) in paymentMethods" :key="i" @click="selected = i" :class="['flex items-center gap-3 p-3 border rounded-lg cursor-pointer transition-all', selected === i ? 'border-primary bg-green-50 border-2' : 'border-slate-200 hover:border-primary/50']">
                <div :class="['w-4.5 h-4.5 rounded-full border-2 flex items-center justify-center', selected === i ? 'border-primary' : 'border-slate-300']">
                  <div v-if="selected === i" class="w-2.5 h-2.5 bg-primary rounded-full"></div>
                </div>
                <div class="flex-1">
                  <p class="text-sm font-medium text-bg-hero">{{ pm.name }}</p>
                  <p class="text-[10px] text-text-dim">{{ pm.desc }}</p>
                </div>
                <div v-if="pm.badge" class="px-2 py-0.5 bg-slate-100 rounded text-[9px] font-bold text-slate-600">{{ pm.badge }}</div>
              </div>
            </div>
            <button class="w-full py-3 bg-primary text-bg-hero font-bold rounded-lg mt-3 hover:bg-primary-hover transition-colors">Finalizar Pagamento</button>
            <div class="flex items-center justify-center gap-1.5 text-[11px] text-text-dim mt-3">
              <svg class="w-3.5 h-3.5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="3" y="11" width="18" height="11" rx="2" stroke-width="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4" stroke-width="2"/></svg>
              Ambiente 100% seguro
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
<script setup>
import { ref } from 'vue'
const selected = ref(0)
const paymentMethods = [
  { name: 'Pix', desc: 'Aprovação instantânea', badge: 'Pix' },
  { name: 'Cartão de Crédito', desc: 'Até 12x sem juros', badge: 'Visa' },
  { name: 'Boleto Bancário', desc: 'Vencimento em 3 dias' }
]
</script>

<template>
  <section class="py-24 bg-bg-hero">
    <div class="container mx-auto px-6">
      <div class="text-center max-w-2xl mx-auto mb-14">
        <h2 class="text-3xl font-bold text-white mb-3">Como funciona</h2>
        <p class="text-text-light">Integre em poucos passos e comece a receber</p>
      </div>
      <div class="grid md:grid-cols-4 gap-5">
        <div v-for="(step, i) in steps" :key="i" class="bg-slate-800 rounded-xl p-6 relative hover:bg-slate-700 transition-colors">
          <div class="w-8 h-8 bg-primary rounded-full flex items-center justify-center text-bg-hero font-bold text-sm mb-4">{{ i + 1 }}</div>
          <h3 class="text-base font-bold text-white mb-2">{{ step.title }}</h3>
          <p class="text-xs text-text-light leading-relaxed">{{ step.desc }}</p>
          <span v-if="i < 3" class="absolute right-[-10px] top-1/2 -translate-y-1/2 text-slate-600 hidden md:block">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>
          </span>
        </div>
      </div>
    </div>
  </section>
</template>
<script setup>
const steps = [
  { title: 'Crie sua conta', desc: 'Cadastre-se gratuitamente e configure suas credenciais de acesso à plataforma em minutos.' },
  { title: 'Integre a API', desc: 'Utilize nossa API RESTful documentada para conectar sua loja com múltiplos gateways de pagamento.' },
  { title: 'Configure pagamentos', desc: 'Ative Pix, cartão de crédito, boleto e outros métodos com split e regras personalizáveis.' },
  { title: 'Comece a vender', desc: 'Receba pagamentos, acompanhe suas vendas em tempo real e escale seu negócio com confiança.' }
]
</script>

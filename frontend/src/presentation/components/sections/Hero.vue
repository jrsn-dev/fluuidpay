<template>
  <section class="hero pt-[140px] pb-[80px]">
    <div class="container mx-auto px-6 grid lg:grid-cols-2 gap-16 items-center">
      <div class="hero-content">
        <div class="inline-flex items-center gap-2 bg-primary/10 border border-primary/20 px-4 py-2 rounded-full text-xs font-bold uppercase tracking-wider text-green-400 mb-6">
          <span class="w-2 h-2 bg-green-400 rounded-full animate-pulse"></span>
          Sistema de Integração de Pagamentos
        </div>
        
        <h1 class="text-5xl lg:text-6xl font-extrabold leading-tight mb-5">
          Integre pagamentos. <span class="text-transparent bg-clip-text bg-gradient-to-r from-primary to-green-400">Venda mais.</span> Cresça sem limites.
        </h1>
        
        <p class="text-lg text-text-light leading-relaxed max-w-xl mb-8">
          Conecte sua loja, gerencie seu negócio e receba pagamentos com segurança através da nossa API simples, poderosa e escalável.
        </p>
        
        <div class="flex flex-wrap gap-4 mb-10">
          <button class="btn-primary">
            Começar Gratuitamente
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 12h14M12 5l7 7-7 7"/></svg>
          </button>
          <button class="btn-outline">Ver Demonstração</button>
        </div>
        
        <div class="flex flex-wrap gap-8 mb-10 text-text-dim">
          <div class="flex flex-col items-center gap-2">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><polyline points="16 18 22 12 16 6" stroke-width="2"/><polyline points="8 6 2 12 8 18" stroke-width="2"/></svg>
            <span class="text-xs">API RESTful</span>
          </div>
          <div class="flex flex-col items-center gap-2">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="2" y="3" width="20" height="14" rx="2" stroke-width="2"/><line x1="8" y1="21" x2="16" y2="21" stroke-width="2"/><line x1="12" y1="17" x2="12" y2="21" stroke-width="2"/></svg>
            <span class="text-xs">Múltiplos Gateways</span>
          </div>
          <div class="flex flex-col items-center gap-2">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><circle cx="9" cy="21" r="1"/><circle cx="20" cy="21" r="1"/><path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6" stroke-width="2"/></svg>
            <span class="text-xs">Carrinho Inteligente</span>
          </div>
          <div class="flex flex-col items-center gap-2">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" stroke-width="2"/><circle cx="9" cy="7" r="4" stroke-width="2"/></svg>
            <span class="text-xs">Gestão de Clientes</span>
          </div>
        </div>
        
        <div class="flex flex-wrap gap-8">
          <div class="flex items-start gap-3">
            <svg class="w-5 h-5 text-primary mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" stroke-width="2"/></svg>
            <div>
              <p class="text-xs font-bold text-text-light">Ambiente 100% Seguro</p>
              <p class="text-[10px] text-text-dim">Conformidade LGPD</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <svg class="w-5 h-5 text-primary mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12" stroke-width="2"/></svg>
            <div>
              <p class="text-xs font-bold text-text-light">Alta Disponibilidade</p>
              <p class="text-[10px] text-text-dim">99.9% Uptime</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <svg class="w-5 h-5 text-primary mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" stroke-width="2"/></svg>
            <div>
              <p class="text-xs font-bold text-text-light">Suporte Especializado</p>
              <p class="text-[10px] text-text-dim">24/7</p>
            </div>
          </div>
        </div>
      </div>
      
      <div class="hero-illustration relative flex justify-center items-center min-h-[480px]">
        <div class="absolute w-1 h-1 bg-primary rounded-full shadow-[0_0_8px_rgba(34,197,94,0.6)] top-[20%] left-[30%] animate-pulse"></div>
        <div class="absolute w-1 h-1 bg-primary rounded-full shadow-[0_0_8px_rgba(34,197,94,0.6)] top-[50%] right-[20%] animate-pulse delay-1000"></div>
        <div class="absolute w-1 h-1 bg-primary rounded-full shadow-[0_0_8px_rgba(34,197,94,0.6)] bottom-[30%] left-[40%] animate-pulse delay-2000"></div>
        <img src="/logo.png" alt="PayIntegra Dashboard" class="max-w-full max-h-[520px] object-contain drop-shadow-[0_20px_40px_rgba(34,197,94,0.15)]" />
      </div>
    </div>
  </section>
</template>

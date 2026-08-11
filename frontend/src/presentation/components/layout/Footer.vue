<template>
  <footer class="py-12 bg-bg-footer border-t border-white/5">
    <div class="container mx-auto px-6">
      <div class="grid md:grid-cols-5 gap-10 mb-10">
        <div class="md:col-span-2">
          <a href="/" class="flex items-center gap-3 text-xl font-bold text-white mb-3">
            <div class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
              <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="#0a0f1c" stroke-width="2.5"><rect x="1" y="4" width="22" height="16" rx="2"/><line x1="1" y1="10" x2="23" y2="10"/></svg>
            </div>
            PayIntegra
          </a>
          <p class="text-sm text-text-dim leading-relaxed mb-4 max-w-xs">Plataforma completa de integração de pagamentos para empresas que querem crescer.</p>
          <span class="text-xs text-slate-600">© 2024 PayIntegra. Todos os direitos reservados.</span>
        </div>
        
        <div v-for="col in columns" :key="col.title" class="footer-col">
          <h4 class="text-sm font-semibold text-white mb-4">{{ col.title }}</h4>
          <ul class="space-y-2.5">
            <li v-for="link in col.links" :key="link.text">
              <a :href="link.href" class="text-sm text-text-light hover:text-white transition-colors">{{ link.text }}</a>
            </li>
          </ul>
        </div>
        
        <div>
          <h4 class="text-sm font-semibold text-white mb-4">Conecte-se</h4>
          <div class="flex gap-2.5">
            <a v-for="social in socials" :key="social.name" :href="social.href" class="w-9 h-9 bg-slate-800 rounded-full flex items-center justify-center text-text-light hover:bg-primary/15 hover:text-primary transition-all">
              <svg class="w-4 h-4" v-html="social.icon"></svg>
            </a>
          </div>
        </div>
      </div>
      
      <div class="border-t border-white/5 pt-5 flex flex-wrap justify-between items-center gap-4">
        <p class="text-xs text-slate-600">© 2024 PayIntegra. Todos os direitos reservados.</p>
        <div class="flex gap-6">
          <a href="#" class="text-xs text-slate-600 hover:text-text-light transition-colors">Política de Privacidade</a>
          <a href="#" class="text-xs text-slate-600 hover:text-text-light transition-colors">LGPD</a>
          <a href="#" class="text-xs text-slate-600 hover:text-text-light transition-colors">Termos de Uso</a>
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup>
const columns = [
  { title: 'Produto', links: [{ text: 'Funcionalidades', href: '#' }, { text: 'Integrações', href: '#' }, { text: 'API Documentation', href: '#' }, { text: 'Preços', href: '#' }] },
  { title: 'Empresa', links: [{ text: 'Sobre nós', href: '#' }, { text: 'Blog', href: '#' }, { text: 'Contato', href: '#' }, { text: 'Status', href: '#' }] },
  { title: 'Suporte', links: [{ text: 'Central de Ajuda', href: '#' }, { text: 'Documentação', href: '#' }, { text: 'Tutoriais', href: '#' }, { text: 'Suporte 24/7', href: '#' }] }
]

const socials = [
  { name: 'GitHub', href: '#', icon: '<path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>' },
  { name: 'LinkedIn', href: '#', icon: '<path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433a2.062 2.062 0 01-2.063-2.065 2.064 2.064 0 112.063 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/>' },
  { name: 'YouTube', href: '#', icon: '<path d="M23.498 6.186a3.016 3.016 0 00-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 00.502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 002.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 002.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z"/>' },
  { name: 'Twitter', href: '#', icon: '<path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>' }
]
</script>

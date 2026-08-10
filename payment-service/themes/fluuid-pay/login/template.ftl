<#macro registrationLayout bodyClass="" displayInfo=false displayMessage=true displayRequiredFields=false>
<!DOCTYPE html>
<html class="kc-html" <#if realm.internationalizationEnabled && locale?? && locale.supported?? && locale.supported?size gt 1> lang="${locale.currentLanguageTag}"</#if>>

<head>
    <meta charset="utf-8">
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <meta name="robots" content="noindex, nofollow">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>${msg("loginTitle",(realm.displayName!''))}</title>
    <link rel="icon" href="${url.resourcesPath}/img/favicon.ico" />
    
    <#if properties.stylesCommon?has_content>
        <#list properties.stylesCommon?split(' ') as style>
            <link href="${url.resourcesCommonPath}/${style}" rel="stylesheet" />
        </#list>
    </#if>
    <#if properties.styles?has_content>
        <#list properties.styles?split(' ') as style>
            <link href="${url.resourcesPath}/${style}" rel="stylesheet" />
        </#list>
    </#if>
</head>

<body class="kc-body">
  <div class="fp-split-layout">
    
    <!-- Lado Esquerdo: Marketing e Branding -->
    <div class="fp-split-left">
      <div class="fp-brand-header">
        <div class="fp-logo-icon"></div>
        <h1>Fluuid Pay</h1>
      </div>
      
      <div class="fp-hero-text">
        <span class="fp-subtitle">SISTEMA DE INTEGRAÇÃO DE PAGAMENTOS</span>
        <h2>Conecte sua loja.<br/>Gerencie seu negócio.<br/><span class="fp-text-accent">Receba mais.</span></h2>
        <p>Plataforma completa para integrar pagamentos, gerenciar clientes, produtos e crescer sem limites com segurança e performance.</p>
      </div>

      <div class="fp-features">
        <div class="fp-feature-item">
          <div class="fp-feature-icon">🛡️</div>
          <div class="fp-feature-content">
            <h3>Ambiente 100% Seguro</h3>
            <p>Seus dados e transações protegidos com criptografia avançada</p>
          </div>
        </div>
        <div class="fp-feature-item">
          <div class="fp-feature-icon">⚡</div>
          <div class="fp-feature-content">
            <h3>Alta Disponibilidade</h3>
            <p>Infraestrutura robusta com 99.9% de uptime garantido</p>
          </div>
        </div>
        <div class="fp-feature-item">
          <div class="fp-feature-icon">🎧</div>
          <div class="fp-feature-content">
            <h3>Suporte Especializado</h3>
            <p>Equipe pronta para ajudar você 24 horas por dia</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Lado Direito: Formulário de Login -->
    <div class="fp-split-right">
      <!-- Elementos decorativos (simulando os gráficos 3D da referência) -->
      <div class="fp-bg-glow fp-glow-1"></div>
      <div class="fp-bg-glow fp-glow-2"></div>
      
      <div class="fp-login-card">
        <div class="fp-card-header">
          <#nested "header">
        </div>
        
        <div class="fp-card-body">
          <#-- Mostrar mensagens de erro/sucesso do Keycloak -->
          <#if displayMessage && message?has_content && (message.type != 'warning' || !isAppInitiatedAction??)>
              <div class="alert alert-${message.type}">
                  <#if message.type = 'success'><span class="kc-feedback-text">${kcSanitize(message.summary)?no_esc}</span></#if>
                  <#if message.type = 'warning'><span class="kc-feedback-text">${kcSanitize(message.summary)?no_esc}</span></#if>
                  <#if message.type = 'error'><span class="kc-feedback-text">${kcSanitize(message.summary)?no_esc}</span></#if>
                  <#if message.type = 'info'><span class="kc-feedback-text">${kcSanitize(message.summary)?no_esc}</span></#if>
              </div>
          </#if>

          <#-- Formulário em si (injetado do login.ftl) -->
          <#nested "form">
          
          <#if displayInfo>
              <div class="fp-card-footer">
                  <#nested "info">
              </div>
          </#if>
        </div>
      </div>
    </div>
  </div>
</body>
</html>
</#macro>

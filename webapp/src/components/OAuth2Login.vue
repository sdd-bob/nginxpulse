<template>
  <div class="oauth2-login-container">
    <div class="login-card">
      <div class="login-header">
        <div class="login-logo-mark">
          <span class="logo-initials">NP</span>
        </div>
        <h1 class="login-title">{{ t('oauth2.welcome') }}</h1>
        <p class="login-subtitle">{{ t('oauth2.subtitle') }}</p>
      </div>
      
      <div class="oauth2-providers">
        <button 
          v-for="provider in providers" 
          :key="provider.name"
          class="oauth2-btn"
          :class="`oauth2-${provider.name}`"
          @click="loginWith(provider.name)"
          type="button"
        >
          <i :class="provider.icon" aria-hidden="true"></i>
          <span>{{ t('oauth2.loginWith', { provider: provider.label }) }}</span>
        </button>
      </div>
      
      <div v-if="allowAccessKey" class="access-key-toggle">
        <button 
          class="toggle-btn" 
          @click="$emit('toggle-access-key')"
          type="button"
        >
          {{ showAccessKeyForm ? t('oauth2.useOAuth2') : t('oauth2.useAccessKey') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { getWebBasePathWithSlash } from '@/utils';

interface Provider {
  name: string;
  label: string;
  icon: string;
}

const props = defineProps<{
  providers?: Provider[];
  allowAccessKey?: boolean;
  showAccessKeyForm?: boolean;
}>();

defineEmits<{
  toggle: [];
  'toggle-access-key': [];
}>();

const { t } = useI18n();

const providers = computed(() => {
  return props.providers || [
    { name: 'github', label: 'GitHub', icon: 'pi pi-github' },
    { name: 'google', label: 'Google', icon: 'pi pi-google' },
  ];
});

function loginWith(providerName: string) {
  const basePath = getWebBasePathWithSlash();
  window.location.href = `${basePath}auth/login?provider=${providerName}`;
}
</script>

<style scoped lang="scss">
.oauth2-login-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-card {
  background: white;
  border-radius: 16px;
  padding: 48px 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  text-align: center;
  max-width: 420px;
  width: 100%;
}

.login-header {
  margin-bottom: 32px;
}

.login-logo-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  margin-bottom: 20px;
  box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
}

.logo-initials {
  color: white;
  font-size: 32px;
  font-weight: 700;
  letter-spacing: 2px;
}

.login-title {
  font-size: 28px;
  font-weight: 700;
  color: #1a1a1a;
  margin: 0 0 8px 0;
}

.login-subtitle {
  font-size: 14px;
  color: #666;
  margin: 0;
}

.oauth2-providers {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 24px;
}

.oauth2-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  width: 100%;
  padding: 14px 24px;
  border: none;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  
  i {
    font-size: 18px;
  }
  
  &.oauth2-github {
    background: #24292e;
    color: white;
    
    &:hover {
      background: #424a4f;
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(36, 41, 46, 0.3);
    }
  }
  
  &.oauth2-google {
    background: #4285f4;
    color: white;
    
    &:hover {
      background: #5b9bff;
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(66, 133, 244, 0.3);
    }
  }
  
  &.oauth2-custom {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    
    &:hover {
      opacity: 0.9;
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
    }
  }
}

.access-key-toggle {
  border-top: 1px solid #e5e5e5;
  padding-top: 20px;
}

.toggle-btn {
  background: none;
  border: none;
  color: #667eea;
  font-size: 14px;
  cursor: pointer;
  padding: 8px 16px;
  border-radius: 6px;
  transition: all 0.2s;
  
  &:hover {
    background: rgba(102, 126, 234, 0.1);
  }
}
</style>

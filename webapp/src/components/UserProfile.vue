<template>
  <div class="user-profile">
    <Dropdown
      v-model="selectedUser"
      :options="userOptions"
      optionLabel="label"
      class="user-dropdown"
      :pt="{
        root: { class: 'user-dropdown-root' },
        input: { class: 'user-dropdown-input' },
        panel: { class: 'user-dropdown-panel' }
      }"
    >
      <template #value="slotProps">
        <div v-if="slotProps.value" class="user-info">
          <i class="pi pi-user"></i>
          <span class="user-email">{{ slotProps.value.label }}</span>
        </div>
      </template>
      <template #option="slotProps">
        <div class="user-option">
          <i class="pi pi-user"></i>
          <span>{{ slotProps.option.label }}</span>
        </div>
      </template>
      <template #footer>
        <div class="dropdown-footer">
          <div class="user-provider" v-if="userProvider">
            <i :class="providerIcon"></i>
            <span>{{ userProvider }}</span>
          </div>
          <Button 
            :label="t('oauth2.logout')" 
            icon="pi pi-sign-out" 
            @click="logout"
            class="p-button-text logout-btn"
          />
        </div>
      </template>
    </Dropdown>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import Dropdown from 'primevue/dropdown';
import Button from 'primevue/button';
import client from '@/api/client';

const { t } = useI18n();

const userEmail = ref<string | null>(null);
const userName = ref<string | null>(null);
const userProvider = ref<string | null>(null);

const userOptions = computed(() => {
  if (!userEmail.value) return [];
  const label = userName.value || userEmail.value;
  return [{ label, value: label }];
});

const selectedUser = ref(null);

const providerIcon = computed(() => {
  switch (userProvider.value) {
    case 'github':
      return 'pi pi-github';
    case 'google':
      return 'pi pi-google';
    default:
      return 'pi pi-user';
  }
});

async function loadUserInfo() {
  try {
    const res = await client.get('/auth/status');
    if (res.data.logged_in) {
      userEmail.value = res.data.email;
      userName.value = res.data.name;
      userProvider.value = res.data.provider;
      selectedUser.value = { label: userName.value || userEmail.value };
    }
  } catch (error) {
    console.error('Failed to load user info:', error);
  }
}

async function logout() {
  try {
    await client.post('/auth/logout');
    window.location.reload();
  } catch (error) {
    console.error('Logout failed:', error);
  }
}

onMounted(() => {
  loadUserInfo();
});
</script>

<style scoped lang="scss">
.user-profile {
  display: flex;
  align-items: center;
}

.user-dropdown {
  min-width: 200px;
}

.user-dropdown-input {
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  
  &:hover, &:focus {
    background: rgba(255, 255, 255, 0.1) !important;
  }
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: white;
  font-size: 14px;
  
  i {
    font-size: 16px;
    opacity: 0.8;
  }
}

.user-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.dropdown-footer {
  padding: 12px;
  border-top: 1px solid #e5e5e5;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.user-provider {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #666;
  padding: 4px 0;
  
  i {
    font-size: 14px;
  }
}

.logout-btn {
  width: 100%;
  justify-content: center;
  color: #dc3545;
  
  &:hover {
    background: rgba(220, 53, 69, 0.1);
  }
}
</style>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const isSubmitting = ref(false)
const errorMessage = ref('')

// Set once on mount from the query string the router guard (or useApi's
// forced logout on a failed refresh) attached — not reactive to later
// query changes, since it should only ever describe how the user arrived
// at this page load.
const sessionExpired = route.query.reason === 'expired'

function redirectTarget(): string {
  const redirect = route.query.redirect
  return typeof redirect === 'string' && redirect.length > 0 ? redirect : '/'
}

async function handleSubmit() {
  errorMessage.value = ''
  isSubmitting.value = true
  try {
    await authStore.login(email.value, password.value)
    await router.push(redirectTarget())
  } catch {
    // Deliberately generic — never reveal whether the email or the
    // password was the one that didn't match.
    errorMessage.value = 'Email o contraseña incorrectos.'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <main class="login">
    <form
      class="login-card"
      @submit.prevent="handleSubmit"
    >
      <h1 class="login-title">
        BrewOps
      </h1>
      <p class="login-subtitle">
        Inicia sesión para continuar
      </p>

      <p
        v-if="sessionExpired"
        class="login-banner login-banner--warning"
        role="alert"
      >
        Tu sesión expiró, ingresa de nuevo.
      </p>
      <p
        v-if="errorMessage"
        class="login-banner login-banner--error"
        role="alert"
      >
        {{ errorMessage }}
      </p>

      <label class="login-field">
        <span>Correo electrónico</span>
        <input
          v-model="email"
          type="email"
          autocomplete="username"
          required
          :disabled="isSubmitting"
        >
      </label>

      <label class="login-field">
        <span>Contraseña</span>
        <input
          v-model="password"
          type="password"
          autocomplete="current-password"
          required
          :disabled="isSubmitting"
        >
      </label>

      <button
        type="submit"
        class="login-submit"
        :disabled="isSubmitting"
      >
        {{ isSubmitting ? 'Ingresando…' : 'Iniciar sesión' }}
      </button>
    </form>
  </main>
</template>

<style scoped>
.login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-4);
  background: var(--color-surface-alt);
}

.login-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  width: 100%;
  max-width: 22rem;
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.login-title {
  margin: 0;
  font-size: 1.5rem;
  color: var(--color-text);
}

.login-subtitle {
  margin: 0;
  color: var(--color-text-muted);
}

.login-banner {
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  font-size: 0.9rem;
}

.login-banner--error {
  color: hsla(4, 75%, 30%, 1);
  background: hsla(4, 75%, 50%, 0.1);
}

.login-banner--warning {
  color: hsla(40, 90%, 28%, 1);
  background: hsla(40, 90%, 50%, 0.15);
}

.login-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  color: var(--color-text);
  font-size: 0.9rem;
}

.login-field input {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 1rem;
  font-family: inherit;
}

.login-field input:disabled {
  background: var(--color-surface-alt);
  color: var(--color-text-muted);
}

.login-submit {
  padding: var(--space-2) var(--space-3);
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: hsla(0, 0%, 100%, 1);
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
}

.login-submit:hover:not(:disabled) {
  background: var(--color-primary-hover);
}

.login-submit:disabled {
  background: var(--color-text-muted);
  cursor: not-allowed;
}
</style>

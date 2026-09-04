/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  // "true" or unset/anything else (default). Gates the demo credentials
  // banner (LoginView) and the general demo notice (AppNav). This repo
  // only; the real product has no equivalent flag.
  readonly VITE_DEMO_MODE?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

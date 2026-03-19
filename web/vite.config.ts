import { loadEnv } from 'vite';
import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const devHost = env.PRACTICEHELPER_WEB_HOST || '0.0.0.0';
  const devPort = Number(env.PRACTICEHELPER_WEB_PORT || '5173');
  const apiProxyTarget =
    env.PRACTICEHELPER_WEB_API_PROXY_TARGET || 'http://127.0.0.1:8090';

  return {
    plugins: [vue()],
    server: {
      host: devHost,
      port: devPort,
      strictPort: true,
      proxy: {
        '/api': {
          target: apiProxyTarget,
          changeOrigin: true,
        },
        '/healthz': {
          target: apiProxyTarget,
          changeOrigin: true,
        },
      },
    },
    test: {
      environment: 'jsdom',
      globals: true,
    },
  };
});

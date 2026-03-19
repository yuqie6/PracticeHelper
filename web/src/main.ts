import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import { createApp } from 'vue';

import App from './App.vue';
import router from './router';
import './style.css';

const queryClient = new QueryClient();
const app = createApp(App);

app.use(router);
app.use(VueQueryPlugin, { queryClient });
app.mount('#app');

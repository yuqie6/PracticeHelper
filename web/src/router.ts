import { createRouter, createWebHistory } from 'vue-router';

import HomeView from './views/HomeView.vue';
import JobTargetsView from './views/JobTargetsView.vue';
import ProfileView from './views/ProfileView.vue';
import ProjectsView from './views/ProjectsView.vue';
import ReviewView from './views/ReviewView.vue';
import SessionView from './views/SessionView.vue';
import TrainView from './views/TrainView.vue';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: HomeView },
    { path: '/profile', component: ProfileView },
    { path: '/job-targets', component: JobTargetsView },
    { path: '/projects', component: ProjectsView },
    { path: '/train', component: TrainView },
    { path: '/sessions/:id', component: SessionView, props: true },
    { path: '/reviews/:id', component: ReviewView, props: true },
  ],
});

export default router;

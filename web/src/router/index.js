import { createRouter, createWebHistory } from "vue-router";
import Index from "../views/IndexView.vue";
import Login from "../views/LoginView.vue";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      alias: ["/index.html"],
      name: "home",
      component: Index,
    },
    {
      path: "/login",
      name: "login",
      component: Login,
    },
    {
      path: "/ranks",
      name: "ranks",
      component: () => import("@/views/RanksView.vue"),
    },
    {
      path: "/events",
      name: "events",
      component: () => import("@/views/EventsView.vue"),
    },
    {
      path: "/error",
      name: "error",
      component: () => import("@/views/ErrorView.vue"),
    },
  ],
});

export default router;

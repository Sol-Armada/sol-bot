import { createRouter, createWebHistory } from "vue-router";
import Index from "../views/IndexView.vue";
import Login from "../views/LoginView.vue";
import Error from "../views/ErrorView.vue";

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
      path: "/admin",
      name: "admin",
      // route level code-splitting
      // this generates a separate chunk (About.[hash].js) for this route
      // which is lazy-loaded when the route is visited.
      component: () => import("@/views/AdminView.vue"),
    },
    {
      path: "/error",
      name: "error",
      component: () => import("@/views/ErrorView.vue")
    }
  ],
});

export default router;

import { createRouter, createWebHistory } from "vue-router";
import { useComposition } from "../compositions";
import cookie from "@point-hub/vue-cookie";

const { admin } = useComposition();

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      name: "dashboard",
      component: () => import("@/views/DashboardView.vue"),
    },
    {
      path: "/login",
      name: "login",
      component: () => import("@/views/LoginView.vue"),
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

router.beforeEach(async (to) => {
  if (admin.value == null && cookie.get("admin") != undefined) {
    admin.value = JSON.parse(cookie.get("admin"));
  }

  if (to.name == "login" && admin.value != null) {
    return { name: "dashboard" };
  }

  if (to.name == "login") {
    return true;
  }

  if (admin.value == null && cookie.get("admin") != undefined) {
    admin.value = JSON.parse(cookie.get("admin"));
  }

  if (admin.value != null) {
    return true;
  }

  return { name: "login" };
});

export default router;

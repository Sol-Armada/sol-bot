<script setup>
import Nav from "./components/NavComponent.vue";
import { RouterView, useRouter } from "vue-router";
import { useComposition } from "./compositions";
import cookie from "@point-hub/vue-cookie";
const { admin } = useComposition();
const router = useRouter();

function logout() {
  admin.value = undefined;
  cookie.remove("admin");
  router.push("/");
}
</script>

<template>
  <div class="nav" v-if="admin">
    <Nav :logout="logout" />
  </div>
  <div class="content">
    <RouterView />
  </div>
</template>

<style scoped>
.content {
  overflow: auto;
  padding: 0px 10px 10px 10px;
  grid-column-start: 2;
  width: 100%;
}

.content:only-child {
  grid-column-start: 1;
  grid-column-end: 3;
  justify-self: center;
  align-self: center;
  text-align: center;
}

.content div {
  border-radius: var(--mdc-shape-small, 4px);
}
</style>

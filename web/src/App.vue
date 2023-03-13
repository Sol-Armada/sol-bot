<script setup>
import Nav from "./components/NavComponent.vue";
import { RouterView, useRouter } from "vue-router";
import { useComposition } from "./compositions";
import cookie from "@point-hub/vue-cookie";
import { getUsers, getBankBalance } from "./api";
import { onMounted } from "vue";

const { admin, users, bank } = useComposition();
const router = useRouter();

function logout() {
  admin.value = undefined;
  cookie.remove("admin");
  router.push("/");
}

onMounted(() => {
  getUsers
    .then((u) => {
      users.value = u;
    })
    .catch((e) => {
      console.log(e);
    });

  getBankBalance
    .then((b) => {
      bank.value = {
        balance: b,
      };
    })
    .catch(console.log);
});
</script>

<template>
  <Nav :logout="logout" v-if="admin" />
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

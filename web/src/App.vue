<script setup>
import Nav from "./components/NavComponent.vue";
import { RouterView, useRouter } from "vue-router";
import { useComposition } from "./compositions";
import cookie from "@point-hub/vue-cookie";
import { getUsers, getBankBalance, getEvents } from "./api";
import { onUpdated } from "vue";

const { admin, users, bank, events } = useComposition();
const router = useRouter();

function logout() {
  admin.value = undefined;
  cookie.remove("admin");
  router.push("/");
}

onUpdated(() => {
  if (admin.value != undefined) {
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

    getEvents
      .then((e) => {
        events.value = e;
      })
      .catch(console.log);
  }
});
</script>

<template>
  <Nav :logout="logout" v-if="admin" />
  <div class="content">
    <RouterView />
  </div>
  <div class="error"></div>
</template>

<style lang="scss" scoped>
.content {
  overflow: auto;
  padding: 0px 10px 10px 10px;
  grid-column-start: 2;
  width: 100%;

  &:only-child {
    grid-column-start: 1;
    grid-column-end: 3;
    justify-self: center;
    align-self: center;
    text-align: center;
  }

  div {
    border-radius: var(--mdc-shape-small, 4px);
  }
}

div.error {
  z-index: 101;
  color: var(--mdc-theme-on-surface);
  background-color: darkred;
  padding: 10px;
  width: fit-content;
  border-radius: 5px;
  position: fixed;
  bottom: -50px;
  left: 10px;
}

.pop-in-and-out {
  animation: pop-in-and-out 5000ms;
}

@keyframes pop-in-and-out {
  0% {
    bottom: -50px;
  }
  10% {
    bottom: 10px;
  }
  90% {
    bottom: 10px;
  }
  100% {
    bottom: -50px;
  }
}
</style>

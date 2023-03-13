<script setup>
import { useRoute, useRouter } from "vue-router";
import { useComposition } from "@/compositions";
import cookie from "@point-hub/vue-cookie";
import axios from "axios";
import { onMounted, ref } from "vue";

const discordAuthUrl = ref(import.meta.env.VITE_DISCORD_AUTH_URL);
const route = useRoute();
const userCode = ref(route.query.code);

onMounted(() => {
  const { err, admin } = useComposition();
  const router = useRouter();
  if (userCode.value != undefined) {
    axios
      .post(
        `${import.meta.env.VITE_API_BASE_URL}/login`,
        {
          code: userCode.value,
        },
        {
          headers: {
            "Content-Type": "application/json",
          },
        }
      )
      .then((resp) => {
        console.log("setting admin");
        admin.value = resp.data.user;
        cookie.set("admin", JSON.stringify(resp.data.user));
        router.push("/");
      })
      .catch((error) => {
        if (error != undefined) {
          if (error.message.includes("401")) {
            err.value = 401;
            router.push("/error");
          } else {
            console.log(error);
          }
        }
      });
  }
});
</script>

<template>
  <h1>Sol Armada Administration Portal</h1>
  <a
    :href="`${discordAuthUrl}`"
    class="mdc-button mdc-button--raised mdc-button--leading"
  >
    <span class="mdc-button__ripple"></span>
    <i class="material-icons mdc-button__icon" aria-hidden="true">discord</i>
    <span class="mdc-button__label">Login with Discord</span>
  </a>
</template>

<style scoped>
h1 {
  color: var(--mdc-theme-on-surface);
  margin-bottom: 10px;
}

.logging-in {
  grid-row-start: 2;
  justify-self: center;
  align-self: center;
  text-align: center;
}

.lds-dual-ring {
  display: inline-block;
  width: 80px;
  height: 80px;
}

.lds-dual-ring:after {
  content: " ";
  display: block;
  width: 64px;
  height: 64px;
  margin: 8px;
  border-radius: 50%;
  border: 6px solid #fff;
  border-color: #fff transparent #fff transparent;
  animation: lds-dual-ring 1.2s linear infinite;
}

@keyframes lds-dual-ring {
  0% {
    transform: rotate(0deg);
  }

  100% {
    transform: rotate(360deg);
  }
}
</style>

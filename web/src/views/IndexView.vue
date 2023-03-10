<script setup>
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useComposition } from "../compositions";
import cookie from "@point-hub/vue-cookie";

const discordAuthUrl = ref(import.meta.env.VITE_DISCORD_AUTH_URL);
const { admin } = useComposition();
const router = useRouter();

if (cookie.get("admin") != undefined && admin.value == undefined) {
  admin.value = JSON.parse(cookie.get("admin"));
  router.push("/ranks");
}
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

<style lang="scss" scoped>
h1 {
  color: var(--mdc-theme-on-surface);
  margin-bottom: 10px;
}
</style>

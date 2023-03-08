<script setup>
import { useRouter } from "vue-router";
import { useComposition } from "@/compositions";
import cookie from "@point-hub/vue-cookie";
import List from "../components/events/ListComponent.vue";
import { onMounted } from "vue";
import { getEvents } from "../api/index";

const { admin, events } = useComposition();
const router = useRouter();

if (cookie.get("admin") != undefined && admin.value == undefined) {
  admin.value = JSON.parse(cookie.get("admin"));
}

if (admin.value == undefined || admin.value.username == "") {
  router.push("/");
}

onMounted(() => {
  const { admin } = useComposition();
  if (admin.value) {
    getEvents();
  }
});
</script>

<template>
  <List :admin="admin" :events="events" />
</template>

<script setup>
import { useRouter } from "vue-router";
import { ref } from "vue";
import { useComposition } from "@/compositions";
import cookie from "@point-hub/vue-cookie";
import List from "../components/events/ListComponent.vue";
import NewEvent from "../components/events/NewEventComponent.vue";

const { admin, events } = useComposition();
const router = useRouter();
const newEvent = ref({
  name: "",
  start: null,
  end: null,
  description: "",
  cover: "",
  auto_start: false,
  positions: new Map(),
});

if (cookie.get("admin") != undefined && admin.value == undefined) {
  admin.value = JSON.parse(cookie.get("admin"));
}

if (admin.value == undefined || admin.value.username == "") {
  router.push("/");
}

function eventCreated(e) {
  e.start = new Date(e.start);
  e.end = new Date(e.end);
  events.value.push(e);
}
</script>

<template>
  <List :admin="admin" :events="events" />
  <NewEvent :event="newEvent" @created="eventCreated" />
</template>

<script setup>
import { useRouter } from "vue-router";
import { useComposition } from "@/compositions";
import cookie from "@point-hub/vue-cookie";
import List from "../components/ListComponent.vue";
import { onMounted } from "vue";
import { updateUser } from "../api/index";
import { getUsers } from "../api/index";

const { admin, users } = useComposition();
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
    getUsers();
  }
});
</script>

<template>
  <List :admin="admin" :users="users" :update-user="updateUser" />
</template>

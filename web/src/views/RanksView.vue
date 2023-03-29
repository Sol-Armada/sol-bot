<script setup>
import { useRouter } from "vue-router";
import { useComposition } from "@/compositions";
import cookie from "@point-hub/vue-cookie";
import List from "../components/ranks/ListComponent.vue";
import { updateUser } from "../api/index";

const { admin, users } = useComposition();
const router = useRouter();

if (cookie.get("admin") != undefined && admin.value == undefined) {
  admin.value = JSON.parse(cookie.get("admin"));
}

if (admin.value == undefined || admin.value.username == "") {
  router.push("/");
}
</script>

<template>
  <List :admin="admin" :users="users" :update-user="updateUser" />
</template>

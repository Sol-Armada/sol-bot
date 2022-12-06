<template>
  <div v-if="err == 401">
    <h1>You are not authorized for this site</h1>
  </div>
  <div v-else>
    <h1>Not Found</h1>
  </div>
  <h3><a @click="goHome()">Return Home</a></h3>
</template>
<script>
import router from "../router";
import { useComposition } from "@/compositions";

export default {
  setup() {
    const { err } = useComposition();
    if (err.value == undefined) {
      router.push("/");
      return;
    }
    return { err };
  },
  methods: {
    goHome() {
      const { err } = useComposition();
      err.value = undefined;
      router.push("/");
    },
  },
};
</script>

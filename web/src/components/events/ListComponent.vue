<script setup>
import Card from "./CardComponent.vue";

defineProps({
  admin: Object,
  events: Array,
});
</script>

<template>
  <div class="list">
    <Card v-for="event in events" :key="event._id" :event="event" />
  </div>
</template>

<style lang="scss" scoped>
@use "@material/fab";

.list {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin: 10px 0;
}
</style>

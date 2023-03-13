<script setup>
import { ref } from "vue";

const id = ref(
  Math.random()
    .toString(36)
    .substring(2, 10 + 2)
);

defineProps({
  position: {
    type: Object,
    default(rawPosition) {
      if (rawPosition != {}) {
        return rawPosition;
      }
      return { name: "", max: 0 };
    },
  },
});
</script>
<template>
  <div :id="'position-' + id" class="position">
    <input
      type="text"
      name="position-name"
      id="position-name"
      placeholder="Position Name"
      :value="position.name"
    />
    <input
      type="number"
      name="position-max"
      id="position-max"
      placeholder="Max"
      :value="position.max"
    />
  </div>
</template>
<style lang="scss" scoped>
.positions {
  flex-wrap: wrap;

  > .position {
    display: grid;
    grid-template-columns: 70% 15%;
    margin: 2px 0;

    > input:last-child {
      grid-column-start: 2;
    }

    > button {
      padding: 0.5vh;
    }
  }
}
</style>

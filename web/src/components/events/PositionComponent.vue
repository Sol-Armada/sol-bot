<script setup>
import { ref } from "vue";

const id = ref(
  Math.random()
    .toString(36)
    .substring(2, 10 + 2)
);

defineProps({
  position: {
    name: String,
    max: Number,
    min_rank: {
      type: Number,
      default() {
        return 8;
      },
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
    <select name="min_rank">
      <option value="99" :selected="position.min_rank >= 99 ? true : false">
        Anyone
      </option>
      <option value="7" :selected="position.min_rank == 7 ? true : false">
        Recruit
      </option>
      <option value="6" :selected="position.min_rank == 6 ? true : false">
        Member
      </option>
      <option value="5" :selected="position.min_rank == 5 ? true : false">
        Technician
      </option>
      <option value="4" :selected="position.min_rank == 4 ? true : false">
        Specialist
      </option>
      <option value="3" :selected="position.min_rank == 3 ? true : false">
        Lieutenant
      </option>
      <option value="2" :selected="position.min_rank == 2 ? true : false">
        Commander
      </option>
      <option value="1" :selected="position.min_rank == 1 ? true : false">
        Admiral
      </option>
    </select>
  </div>
</template>
<style lang="scss" scoped>
.position {
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
.position {
  display: grid;
  grid-template-columns: 60% 15% 25%;
  margin: 2px 0;

  > button {
    padding: 0.5vh;
  }
}
</style>

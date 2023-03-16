<script setup>
import { ref, onUpdated } from "vue";

const id = ref(
  Math.random()
    .toString(36)
    .substring(2, 10 + 2)
);

const props = defineProps({
  position: {
    name: {
      type: String,
      default() {
        return "";
      },
    },
    max: {
      type: Number,
      default() {
        return null;
      },
    },
    min_rank: {
      type: Number,
      default() {
        return 8;
      },
    },
  },
});

const nposition = ref(props.position);

onUpdated(() => {
  if (typeof nposition.value.min_rank == "string") {
    nposition.value.min_rank = parseInt(nposition.value.min_rank);
  }
});
</script>
<template>
  <div :id="'position-' + id" class="position">
    <input
      type="text"
      :name="'position-name-' + id"
      :id="'position-name-' + id"
      placeholder="Position Name"
      v-model="nposition.name"
    />
    <input
      type="number"
      name="position-max"
      id="position-max"
      placeholder="Max"
      v-model="nposition.max"
    />
    <select name="min_rank" v-model="nposition.min_rank">
      <option value="99" :selected="nposition.min_rank >= 99 ? true : false">
        Anyone
      </option>
      <option value="7" :selected="nposition.min_rank == 7 ? true : false">
        Recruit
      </option>
      <option value="6" :selected="nposition.min_rank == 6 ? true : false">
        Member
      </option>
      <option value="5" :selected="nposition.min_rank == 5 ? true : false">
        Technician
      </option>
      <option value="4" :selected="nposition.min_rank == 4 ? true : false">
        Specialist
      </option>
      <option value="3" :selected="nposition.min_rank == 3 ? true : false">
        Lieutenant
      </option>
      <option value="2" :selected="nposition.min_rank == 2 ? true : false">
        Commander
      </option>
      <option value="1" :selected="nposition.min_rank == 1 ? true : false">
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

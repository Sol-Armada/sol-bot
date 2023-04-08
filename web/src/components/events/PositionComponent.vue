<script setup>
import { ref, onUpdated } from "vue";
import EmojiConvertor from "emoji-js";

var emoji = new EmojiConvertor();
emoji.img_sets.apple.path = "/emojis/";

const props = defineProps({
  position: Object,
  removePos: Function,
});

const positionRef = ref(props.position);

function removeSelf(id) {
  console.log(id);
  document.querySelector("#position-" + id).remove();
  props.removePos(id);
}

onUpdated(() => {
  positionRef.value.emojiconv = "";
  if (positionRef.value.emoji != "") {
    positionRef.value.emojiconv = emoji.replace_colons(
      ":" + positionRef.value.emoji.toLowerCase() + ":"
    );
  }
  if (typeof positionRef.value.min_rank == "string") {
    positionRef.value.min_rank = parseInt(positionRef.value.min_rank);
  }
});
</script>
<template>
  <div :id="'position-' + position.id" class="position">
    <input
      type="text"
      name="position-emoji"
      :id="'position-emoji-' + position.id"
      placeholder="Emoji"
      v-model="positionRef.emoji"
    />
    <input
      type="text"
      :name="'position-name-' + position.id"
      :id="'position-name-' + position.id"
      placeholder="Position Name"
      v-model="positionRef.name"
    />
    <input
      type="number"
      :name="'position-max-' + position.id"
      :id="'position-max-' + position.id"
      placeholder="Max"
      v-model="positionRef.max"
    />
    <select name="min_rank" v-model="positionRef.min_rank">
      <option value="99" :selected="positionRef.min_rank >= 99 ? true : false">
        Anyone
      </option>
      <option value="7" :selected="positionRef.min_rank == 7 ? true : false">
        Recruit
      </option>
      <option value="6" :selected="positionRef.min_rank == 6 ? true : false">
        Member
      </option>
      <option value="5" :selected="positionRef.min_rank == 5 ? true : false">
        Technician
      </option>
      <option value="4" :selected="positionRef.min_rank == 4 ? true : false">
        Specialist
      </option>
      <option value="3" :selected="positionRef.min_rank == 3 ? true : false">
        Lieutenant
      </option>
      <option value="2" :selected="positionRef.min_rank == 2 ? true : false">
        Commander
      </option>
      <option value="1" :selected="positionRef.min_rank == 1 ? true : false">
        Admiral
      </option>
    </select>
    <button v-on:click="removeSelf(positionRef.id)">
      <span class="material-symbols-outlined"> delete </span>
    </button>
  </div>
</template>
<style lang="scss" scoped>
.position {
  display: grid;
  grid-template-columns: 10% auto 8% 15% 8%;
  justify-content: center;
  margin: 2px 0;
  flex-basis: 100%;
  .material-symbols-outlined {
    font-variation-settings: "FILL" 0, "wght" 400, "GRAD" 0, "opsz" 24;
  }

  > input[name="position-emoji"] {
    text-transform: lowercase;
  }
}
</style>

<script setup>
import { onMounted } from "vue";
import { MDCTextField } from "@material/textfield";

const props = defineProps({
  position: Object,
});

onMounted(() => {
  var textField = new MDCTextField(document.querySelector(".mdc-text-field"));
  textField.value = props.position.name;
});
</script>
<template>
  <div class="position">
    <label class="mdc-text-field mdc-text-field--outlined">
      <span class="mdc-notched-outline">
        <span class="mdc-notched-outline__leading"></span>
        <span class="mdc-notched-outline__notch">
          <span class="mdc-floating-label" id="position-name-label">
            Position Name
          </span>
        </span>
        <span class="mdc-notched-outline__trailing"></span>
      </span>
      <input
        class="mdc-text-field__input"
        type="text"
        name="position-name"
        required
      />
    </label>
    <input type="number" placeholder="Max" name="position-count" required />
    <button class="mdc-icon-button mdc-icon-button--raised">
      <div class="mdc-icon-button__ripple"></div>
      <span class="mdc-icon-button__focus-ring"></span>
      <i class="material-icons">delete</i>
    </button>
  </div>
</template>
<style lang="scss" scoped>
@use "@material/button";
@use "@material/floating-label/mdc-floating-label";
@use "@material/line-ripple/mdc-line-ripple";
@use "@material/notched-outline/mdc-notched-outline";
@use "@material/textfield";

.position {
  display: grid;
  grid-template-columns: 70% 15% 15%;
  margin: 2px 0;

  > input:last-child {
    grid-column-start: 2;
  }

  > button {
    padding: 0.5vh;
  }
}
</style>

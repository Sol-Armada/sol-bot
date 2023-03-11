<script setup>
import Position from "./PositionComponent.vue";

defineProps({
  event: Object,
});

// function addPosition() {
// e.preventDefault();
// if (positionCount.value <= 5) {
//   var positions = document.querySelector(".positions");

//   var newPosition = document.createElement("div");
//   newPosition.classList.add("position");
//   var newPositionName = document.createElement("input");
//   newPositionName.type = "text";
//   newPositionName.placeholder = "Position Name";
//   newPositionName.name = "position-name";
//   newPositionName.required = true;
//   var newPositionCount = document.createElement("input");
//   newPositionCount.type = "number";
//   newPositionCount.placeholder = "Max";
//   newPositionCount.name = "position-count";
//   newPositionCount.required = true;
//   var removePositionBtn = document.createElement("button");
//   removePositionBtn.innerText = "delete";
//   removePositionBtn.classList.add("material-symbols-outlined");
//   removePositionBtn.addEventListener("click", () => {
//     newPosition.remove();
//     positionCount.value--;
//   });

//   newPosition.appendChild(newPositionName);
//   newPosition.appendChild(newPositionCount);
//   newPosition.appendChild(removePositionBtn);

//   positions.appendChild(newPosition);

//   positionCount.value++;
// }
// }
</script>
<template>
  <div id="edit" class="modal" v-if="!isHidden" v-on:click="hideModal">
    <div>
      <form v-on:submit="newEvent">
        <label class="mdc-text-field mdc-text-field--outlined">
          <span class="mdc-notched-outline">
            <span class="mdc-notched-outline__leading"></span>
            <span class="mdc-notched-outline__notch">
              <span class="mdc-floating-label" id="name-label"> Name </span>
            </span>
            <span class="mdc-notched-outline__trailing"></span>
          </span>
          <input
            class="mdc-text-field__input"
            type="text"
            name="name"
            required
          />
        </label>
        <div>
          <label for="name">Name: </label>
          <input
            type="text"
            name="name"
            id="name"
            :value="event.name"
            required
          />
        </div>
        <div>
          <label for="start">Start: </label>
          <input type="datetime-local" name="start" id="start" required />
        </div>
        <div>
          <label for="end">End: </label>
          <input type="time" name="end" id="end" required />
        </div>
        <div class="break"></div>
        <div>
          <textarea
            name="description"
            id="description"
            cols="45"
            rows="10"
            placeholder="Description of the event"
            :value="event.description"
          ></textarea>
        </div>
        <div class="break"></div>
        <div>
          <label for="cover">Header Image URL: </label>
          <input type="url" name="cover" id="cover" :value="event.cover" />
        </div>
        <div class="break"></div>
        <div>
          <label for="auto-start">Auto Start: </label>
          <input
            type="checkbox"
            name="auto-start"
            id="auto-start"
            :checked="event.auto_start"
          />
        </div>
        <div class="break"></div>
        <div class="positions">
          <Position
            v-for="(position, index) in event.positions"
            :key="index"
            :position="position"
          />
        </div>
        <div class="button-wrapper">
          <button class="add-position" v-on:click="addPosition">
            Add Position
          </button>
        </div>
        <div class="button-wrapper">
          <button type="submit">Create</button>
        </div>
      </form>
    </div>
  </div>
</template>

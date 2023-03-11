<script setup>
import { ref } from "vue";
import { createEvent } from "../../api/index";

const show = ref(false);

function hideModal(e) {
  var modal = document.querySelector(".modal>div");
  var rect = modal.getBoundingClientRect();

  if (
    e.x <= rect.left ||
    e.x >= rect.right ||
    e.y <= rect.top ||
    e.y >= rect.bottom
  ) {
    show.value = false;
  }
}

function newEvent(e) {
  e.preventDefault();

  var name = document.getElementById("name");
  var start = document.getElementById("start");
  var startEdited = start.value + ":00.000Z";
  var end = document.getElementById("end");
  var endEdited = startEdited.split("T")[0] + "T" + end.value + ":00.000Z";
  var autoStart = document.getElementById("auto-start");
  var autoStartBool = autoStart == "on" ? true : false;
  var description = document.getElementById("description");
  var cover = document.getElementById("cover");

  var positions = document.querySelectorAll(".position");
  var positionsMap = {};
  positions.forEach((position) => {
    if (position.children[0].value != "") {
      positionsMap[position.children[0].value] = parseInt(
        position.children[1].value
      );
    }
  });

  createEvent(
    name.value,
    startEdited,
    endEdited,
    autoStartBool,
    positionsMap,
    description.value,
    cover.value
  );

  show.value = false;
  name.value = "";
  start.value = "";
  end.value = "";
  autoStart.value = true;
  description.value = "";
  cover.value = "";
  positions.forEach((position) => {
    position.children[0].value = "";
    position.children[1].value = "";
  });
}
</script>
<template>
  <div class="modal" v-on:click="hideModal" v-if="show">
    <div>
      <h1>New Event</h1>
      <form v-on:submit="newEvent">
        <div>
          <label for="name">Name: </label>
          <input type="text" name="name" id="name" required />
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
          ></textarea>
        </div>
        <div class="break"></div>
        <div>
          <label for="cover">Header Image URL: </label>
          <input type="url" name="cover" id="cover" />
        </div>
        <div class="break"></div>
        <div>
          <label for="auto-start">Auto Start: </label>
          <input type="checkbox" name="auto-start" id="auto-start" checked />
        </div>
        <div class="break"></div>
        <div class="positions">
          <div class="position" v-for="n in 6" :key="n" id="position-{{ n }}">
            <input
              type="text"
              name="position-name-{{n}}"
              id="position-name-{{n}}"
              placeholder="Position Name"
            />
            <input
              type="number"
              name="position-max-{{n}}"
              id="position-max-{{n}}"
              placeholder="Max"
            />
          </div>
        </div>
        <div class="button-wrapper">
          <button type="submit">Create</button>
        </div>
      </form>
    </div>
  </div>
  <button
    class="new-event-btn mdc-fab mdc-fab--extended"
    v-on:click="show = true"
  >
    <div class="mdc-fab__ripple"></div>
    <span class="material-icons mdc-fab__icon">add</span>
    <span class="mdc-fab__label">Create</span>
  </button>
</template>
<style lang="scss" scoped>
.modal {
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  position: absolute;
  top: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;

  > div {
    color: black;
    position: absolute;
    max-width: 500px;
    min-height: 220px;
    background-color: lightgray;
    grid-template-rows: 25% 75%;

    > h1 {
      background-color: grey;
      width: 100%;
      text-align: center;
    }

    > form {
      display: flex;
      margin: 10px;
      flex-wrap: wrap;
      align-items: center;
      justify-content: center;

      > div:first-child {
        width: 100%;
      }

      > div {
        display: flex;
        justify-content: center;
        align-items: center;
        margin: 2px;

        > input {
          margin: 5px;
        }

        > label {
          margin: 5px;
        }
      }

      > .positions {
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

      > .button-wrapper {
        display: flex;
        grid-column-start: 1;
        grid-column-end: 3;
        align-items: center;
        justify-content: center;
        width: 100%;

        > button {
          min-width: 55px;
          padding: 5px 20px;
        }
      }
    }
  }
}

.new-event-btn {
  position: fixed;
  bottom: 20px;
  right: 35px;
}
</style>

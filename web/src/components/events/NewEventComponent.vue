<script setup>
import { ref } from "vue";
import { createEvent } from "../../api/index";
import Position from "./PositionComponent.vue";
import { useComposition } from "../../compositions";

const show = ref(false);
const { events } = useComposition();

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
  var startDate = new Date(start.value);
  var end = document.getElementById("end");
  var endDate = new Date(start.value.slice(0, 10) + "T" + end.value);

  if (startDate >= endDate) {
    endDate.setDate(endDate.getDate() + 1);
  }

  var autoStart = document.getElementById("auto-start");
  var autoStartBool = autoStart == "on" ? true : false;
  var description = document.getElementById("description");
  var cover = document.getElementById("cover");

  var positions = document.querySelectorAll(".position");
  var positionsMap = {};
  positions.forEach((position) => {
    if (position.children[0].value != "") {
      positionsMap[position.children[0].value] = {
        name: position.children[0].value,
        max: parseInt(position.children[1].value),
        min_rank: parseInt(position.children[2].value),
      };
    }
  });

  var newEvent = {
    name: name.value,
    start: startDate.toISOString(),
    end: endDate.toISOString(),
    auto_start: autoStartBool,
    positions: positionsMap,
    description: description.value,
    cover: cover.value,
  };

  createEvent(newEvent).then((newEvent) => {
    events.value.push(newEvent);
  });

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
          <Position v-for="n in 6" :key="n" :position="[]" />
          <!-- <div class="position" v-for="n in 6" :key="n" id="position-{{ n }}">
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
            <select name="max-rank">
              <option value="99">Min Rank</option>
              <option value="6">Recruit</option>
              <option value="5">Member</option>
              <option value="4">Technician</option>
              <option value="3">Specialist</option>
              <option value="2">Lieutenant</option>
              <option value="1">Commander</option>
              <option value="0">Admiral</option>
            </select>
          </div> -->
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
  position: fixed;
  top: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;

  > div {
    color: var(--mdc-theme-on-surface);
    position: absolute;
    max-width: 500px;
    min-height: 220px;
    background-color: var(--mdc-theme-surface);
    grid-template-rows: 25% 75%;

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

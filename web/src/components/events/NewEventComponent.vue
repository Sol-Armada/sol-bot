<script setup>
import { ref } from "vue";
import { createEvent } from "../../api/index";
import Position from "./PositionComponent.vue";
import { useComposition } from "../../compositions";

const props = defineProps({
  event: Object,
});

const show = ref(false);
const nevent = ref(props.event);
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

  var start = document.getElementById("start");
  var startDate = new Date(start.value);
  var end = document.getElementById("end");
  var endDate = new Date(start.value.slice(0, 10) + "T" + end.value);

  if (startDate >= endDate) {
    endDate.setDate(endDate.getDate() + 1);
  }

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

  nevent.value.start = startDate.toISOString();
  nevent.value.end = endDate.toISOString();
  nevent.value.auto_start = document.getElementById("auto-start") == "on";
  nevent.value.positiions = positions;

  createEvent(nevent.value).then((newEvent) => {
    
    events.value.push(newEvent);
  });

  show.value = false;

  nevent.value = {
    name: "",
    start: null,
    end: null,
    description: "",
    cover: "",
    auto_start: false,
    positions: [
      {
        name: "",
        max: null,
        min_rank: 99,
      },
      {
        name: "",
        max: null,
        min_rank: 99,
      },
      {
        name: "",
        max: null,
        min_rank: 99,
      },
      {
        name: "",
        max: null,
        min_rank: 99,
      },
      {
        name: "",
        max: null,
        min_rank: 99,
      },
      {
        name: "",
        max: null,
        min_rank: 99,
      },
    ],
  };
}
</script>
<template>
  <div class="modal" v-on:click="hideModal" v-if="show">
    <div>
      <form v-on:submit="newEvent">
        <div>
          <label for="name">Name: </label>
          <input
            type="text"
            name="name"
            id="name"
            v-model="nevent.name"
            required
          />
        </div>
        <div>
          <label for="start">Start: </label>
          <input
            type="datetime-local"
            name="start"
            id="start"
            v-model="nevent.start"
            required
          />
        </div>
        <div>
          <label for="end">End: </label>
          <input
            type="time"
            name="end"
            id="end"
            v-model="nevent.end"
            required
          />
        </div>
        <div class="break"></div>
        <div>
          <textarea
            name="description"
            id="description"
            cols="45"
            rows="10"
            placeholder="Description of the event"
            v-model="nevent.description"
          ></textarea>
        </div>
        <div class="break"></div>
        <div>
          <label for="cover">Header Image URL: </label>
          <input type="url" name="cover" id="cover" v-model="nevent.cover" />
        </div>
        <div class="break"></div>
        <div>
          <label for="auto-start">Auto Start: </label>
          <input
            type="checkbox"
            name="auto-start"
            id="auto-start"
            v-model="nevent.auto_start"
          />
        </div>
        <div class="break"></div>
        <div class="positions">
          <Position v-for="(p, i) in nevent.positions" :key="i" :position="p" />
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

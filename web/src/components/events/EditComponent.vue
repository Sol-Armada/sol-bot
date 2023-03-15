<script setup>
import { ref, onUpdated } from "vue";
import Position from "./PositionComponent.vue";
import { updateEvent as ue } from "../../api/index";

const show = ref(false);

const props = defineProps({
  event: Object,
});

const event = ref(props.event);

onUpdated(() => {
  if (show.value == true) {
    var now = new Date();
    var startEle = document.getElementById("start");
    var startDate = new Date(props.event.start);
    startEle.value = new Date(
      startDate.getTime() + now.getTimezoneOffset() * 60000
    )
      .toISOString()
      .slice(0, 16);

    var endEle = document.getElementById("end");
    var endDate = new Date(props.event.end);
    endEle.value = new Date(endDate.getTime() + now.getTimezoneOffset() * 60000)
      .toISOString()
      .slice(11, 16);
  }
});

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

function updateEvent(e) {
  e.preventDefault();

  var name = document.getElementById("name");
  var start = document.getElementById("start");
  var startDate = new Date(start.value);
  var end = document.getElementById("end");
  var endDate = new Date(start.value.slice(0, 10) + "T" + end.value);

  if (startDate > endDate) {
    endDate.setDate(endDate.getDate() + 1);
  }

  var autoStart = document.getElementById("auto-start");
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

  event.value.name = name.value;
  event.value.start = startDate.toISOString();
  event.value.event = endDate.toISOString();
  event.value.auto_start = autoStart == "on" ? true : false;
  event.value.positionsMap = positionsMap;
  event.value.description = description.value;
  event.value.cover = cover.value;

  ue(event.value).then((updatedEvent) => {
    event.value = updatedEvent;
  });
}

defineExpose({ show });
</script>
<template>
  <div
    :id="'modal-' + event._id"
    class="modal"
    v-on:click="hideModal"
    v-if="show"
  >
    <div>
      <form v-on:submit="updateEvent">
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
            v-for="n in 6"
            :key="n"
            :position="
              event.positions[n - 1] == null ? [] : event.positions[n - 1]
            "
          />
        </div>
        <div class="button-wrapper">
          <button type="submit">Update</button>
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
    position: absolute;
    max-width: 500px;
    min-height: 220px;
    background-color: var(--mdc-theme-surface);
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

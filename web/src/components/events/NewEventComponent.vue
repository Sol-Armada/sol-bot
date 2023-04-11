<script setup>
import { ref, watch } from "vue";
import { createEvent, getRandomNames } from "../../api/index";
import Position from "./PositionComponent.vue";
import { error } from "../../utils";

const props = defineProps({
  event: Object,
});
const emit = defineEmits(["created"]);

const showRef = ref(false);
const eventRef = ref(props.event);

function hideModal(e) {
  var modal1 = document.querySelector(".modal>div");
  var rect1 = modal1.getBoundingClientRect();
  var modal2 = document.querySelector(".modal .preview");
  var rect2 = modal2.getBoundingClientRect();

  if (
    e.x <= rect1.left ||
    e.x >= rect2.right ||
    e.y <= rect1.top ||
    e.y >= rect1.bottom
  ) {
    showRef.value = false;
  }
}

function newEvent(e) {
  e.preventDefault();
  const startDate = new Date(eventRef.value.start);
  const endDate = new Date(
    eventRef.value.start.slice(0, 10) + "T" + eventRef.value.end
  );

  if (startDate >= endDate) {
    endDate.setDate(endDate.getDate() + 1);
  }

  if (startDate <= new Date()) {
    return;
  }

  var newEvent = eventRef.value;

  newEvent.start = startDate.toISOString();
  newEvent.end = endDate.toISOString();

  createEvent(newEvent)
    .then((createdEvent) => {
      showRef.value = false;

      emit("created", createdEvent);

      eventRef.value = {
        name: "",
        start: null,
        end: null,
        description: "",
        cover: "",
        auto_start: false,
        positions: new Map(),
      };
    })
    .catch((err) => {
      if (err.response.data == "event overlaps existing event") {
        error("You can not create an event that overlaps another");
      }
    });
}

function addPosition(e) {
  e.preventDefault();
  const id = Math.random()
    .toString(36)
    .substring(2, 10 + 2);
  eventRef.value.positions.set(id, {
    id: id,
    emoji: "",
    name: "",
    max: 1,
    min_rank: 99,
    _names: "",
  });
}

function removePos(id) {
  eventRef.value.positions.delete(id);
}

watch(eventRef.value.positions, () => {
  eventRef.value.positions.forEach((p) => {
    if (p._names == "" || (p.max > 0 && p._names.split("\n").length != p.max)) {
      getRandomNames(p.max, p.min_rank).then((names) => {
        p._names = names;
      });
    }
  });
});
</script>
<template>
  <div class="modal" v-on:click="hideModal" v-if="showRef">
    <div>
      <form v-on:submit="newEvent">
        <div>
          <label for="name">Name: </label>
          <input
            type="text"
            name="name"
            id="name"
            v-model="eventRef.name"
            required
          />
        </div>
        <div>
          <label for="start">Start: </label>
          <input
            type="datetime-local"
            name="start"
            id="start"
            v-model="eventRef.start"
            :min="new Date().toLocaleString().slice(0, 16)"
            required
          />
        </div>
        <div>
          <label for="end">End: </label>
          <input
            type="time"
            name="end"
            id="end"
            v-model="eventRef.end"
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
            v-model="eventRef.description"
          ></textarea>
        </div>
        <div class="break"></div>
        <div>
          <label for="cover">Header Image URL: </label>
          <input
            type="url"
            name="cover"
            id="cover"
            v-model="eventRef.cover"
            placeholder="Defaults to logo"
          />
        </div>
        <div class="break"></div>
        <!-- <div>
          <label for="auto-start">Auto Start: </label>
          <input
            type="checkbox"
            name="auto-start"
            id="auto-start"
            v-model="eventRef.auto_start"
          />
        </div> -->
        <div class="break"></div>
        <div class="positions">
          <Position
            v-for="[id, p] in eventRef.positions"
            :key="id"
            :position="p"
            :removePos="removePos"
          />
        </div>
        <button v-on:click="addPosition">Add</button>
        <div class="button-wrapper">
          <button type="submit">Create</button>
        </div>
      </form>
    </div>
    <div class="preview">
      <div class="embed-grid">
        <div class="grid-title">{{ eventRef.name }}</div>
        <div class="grid-description">{{ eventRef.description }}</div>
        <div class="embed-fields">
          <div class="embed-field">
            <div class="embed-field-name">Time</div>
            <div class="embed-field-value">
              <span class="timestamp">{{ eventRef.start }}</span>
              -
              <span class="timestamp">{{ eventRef.end }}</span>
            </div>
          </div>

          <div
            class="embed-field embed-field-inline"
            v-for="[k, p] in eventRef.positions"
            :key="k"
          >
            <div class="embed-field-name">
              <span v-if="p.emojiconv" v-html="p.emojiconv"></span>
              {{ p.name }} <span v-if="p.max">({{ p.max }}/{{ p.max }})</span>
            </div>
            <div class="embed-field-value">
              <div class="blockquote-container">
                <div class="blockquote-divider"></div>
                <blockquote>{{ p._names }}</blockquote>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  <button
    class="new-event-btn mdc-fab mdc-fab--extended"
    v-on:click="showRef = true"
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
  justify-content: left;
  z-index: 100;
  padding-left: 275px;

  > div {
    color: var(--mdc-theme-on-surface);
    max-width: 500px;
    background-color: var(--mdc-theme-surface);

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

  > div.preview {
    margin-left: 10px;
    display: flex;

    > .embed-grid {
      border-left: 4px solid black;
      overflow: hidden;
      padding: 0.5rem 1rem 1rem 0.75rem;
      display: grid;
      grid-template-columns: auto;
      grid-template-rows: auto;
      max-width: 516px;

      > .grid-title {
        min-width: 0;
        color: fff;
        font-size: 1rem;
        font-weight: 600;
        display: inline-block;
        grid-column: 1/1;
      }

      > .embed-fields {
        min-width: 0;
        display: flex;
        margin-top: 8px;
        flex-wrap: wrap;

        > .embed-field.embed-field-inline {
          flex-basis: 33.33333%;
        }
        > .embed-field {
          font-size: 0.875rem;
          line-height: 1.125rem;
          min-width: 0;
          font-weight: 400;
          flex-basis: 100%;
          flex-grow: 1;
          margin-bottom: 8px;

          > .embed-field-name {
            font-weight: 600;
            margin-bottom: 2px;
            font-size: 0.875rem;
            line-height: 1.125rem;
            min-width: 0;
          }

          > .embed-field-value {
            font-size: 0.875rem;
            line-height: 1.125rem;
            font-weight: 400;
            white-space: pre-line;
            min-width: 0;

            > .blockquote-container {
              display: flex;

              > .blockquote-divider {
                width: 4px;
                border-radius: 4px;
                background-color: grey;
              }

              > blockquote {
                max-width: 100%;
                padding: 0 8px 0 12px;
                box-sizing: border-box;
                text-indent: 0;
                text-overflow: ellipsis;

                white-space: pre-wrap;
              }
            }
          }
        }
      }
    }
  }
}

.timestamp {
  background-color: rgba(78, 80, 88, 0.48);
}

.new-event-btn {
  position: fixed;
  bottom: 20px;
  right: 35px;
  z-index: 100;
}
</style>

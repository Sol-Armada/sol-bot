<script setup>
// import { ref } from "vue";
// import { createEvent } from "../../api";

defineProps({
  admin: Object,
  events: Array,
});

// const isHidden = ref("isHidden");

// var positionCount = 0;

// function hideModal(e) {
//   var modal = document.querySelector(".modal>div");
//   var rect = modal.getBoundingClientRect();

//   if (
//     e.x <= rect.left ||
//     e.x >= rect.right ||
//     e.y <= rect.top ||
//     e.y >= rect.bottom
//   ) {
//     isHidden.value = true;
//   }
// }

// function addPosition(e) {
//   e.preventDefault();
//   if (positionCount <= 5) {
//     var positions = document.querySelector(".positions");

//     var newPosition = document.createElement("div");
//     newPosition.classList.add("position");
//     var newPositionName = document.createElement("input");
//     newPositionName.type = "text";
//     newPositionName.placeholder = "Position Name";
//     newPositionName.name = "position-name";
//     newPositionName.required = true;
//     var newPositionCount = document.createElement("input");
//     newPositionCount.type = "number";
//     newPositionCount.placeholder = "Max";
//     newPositionCount.name = "position-count";
//     newPositionCount.required = true;
//     var removePositionBtn = document.createElement("button");
//     removePositionBtn.innerText = "delete";
//     removePositionBtn.classList.add("material-symbols-outlined");
//     removePositionBtn.addEventListener("click", () => {
//       newPosition.remove();
//       positionCount--;
//     });

//     newPosition.appendChild(newPositionName);
//     newPosition.appendChild(newPositionCount);
//     newPosition.appendChild(removePositionBtn);

//     positions.appendChild(newPosition);

//     positionCount++;
//   }
// }

// function newEvent(e) {
//   e.preventDefault();

//   var name = document.getElementById("name").value;
//   var start = document.getElementById("start").value;
//   start = start + ":00.000Z";
//   var end = document.getElementById("end").value;
//   end = start.split("T")[0] + "T" + end + ":00.000Z";
//   var autoStart = document.getElementById("auto-start").value;
//   autoStart = autoStart == "on" ? true : false;
//   var description = document.getElementById("description").value;
//   var cover = document.getElementById("cover").value;

//   var positions = document.querySelectorAll(".position");
//   var positionsMap = {};
//   positions.forEach((position) => {
//     positionsMap[position.children[0].value] = parseInt(
//       position.children[1].value
//     );
//   });

//   createEvent(name, start, end, autoStart, positionsMap, description, cover);

//   isHidden.value = true;
// }
</script>

<template>
  <div class="list">
    <div
      class="event mdc-card"
      v-for="event in events"
      :key="event._id"
      :id="event._id"
    >
      <div
        class="my-card__media mdc-card__media mdc-card__media--16-9"
        :style="{ backgroundImage: 'url(' + event.cover + ')' }"
      ></div>
      <div class="mdc-card-wrapper__text-section">
        <div class="event__title">{{ event.name }}</div>
        <div class="event__subhead">{{ event._schedule }}</div>
      </div>
      <div class="mdc-card-wrapper__text-section">
        <div class="event__supporting-text">
          {{ event.description }}
        </div>
      </div>
      <div class="mdc-card__actions">
        <button
          class="mdc-button mdc-button--leading mdc-card__action mdc-card__action--icon"
          title="Edit"
        >
          <span class="mdc-button__ripple"></span>
          <i class="material-icons mdc-button__icon" aria-hidden="true">edit</i>
          <span class="mdc-button__label">Edit</span>
        </button>
        <button
          class="delete-icon-button mdc-button mdc-button--raised mdc-button--leading mdc-card__action mdc-card__action--icon"
          title="Delete"
        >
          <span class="mdc-button__ripple"></span>
          <i class="material-icons mdc-button__icon" aria-hidden="true">
            delete
          </i>
          <span class="mdc-button__label">Delete</span>
        </button>
      </div>
    </div>
  </div>
  <!-- <div class="list">
    <div class="card new" v-on:click="isHidden = false">
      <div>
        <span class="material-symbols-outlined"> add_box </span>
      </div>
    </div>
    <div
      v-for="event in events"
      :key="event._id"
      class="card event"
      :id="event._id"
    >
      <div
        class="cover"
        :style="{ backgroundImage: 'url(' + event.cover + ')' }"
      ></div>
      <div>
        <div class="title">{{ event.name }}</div>
        <div class="time">{{ event.start_date }}</div>
      </div>
    </div>
  </div>
  <div class="modal" v-if="!isHidden" v-on:click="hideModal">
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
        <div class="positions"></div>
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
  </div> -->
</template>

<style lang="scss" scoped>
@use "@material/card";
@use "@material/button";

// @include card.core-styles;
// .modal {
//   width: 100%;
//   height: 100%;
//   background-color: rgba(0, 0, 0, 0.5);
//   position: absolute;
//   top: 0;
//   left: 0;
//   display: flex;
//   align-items: center;
//   justify-content: center;

//   > div {
//     color: black;
//     position: absolute;
//     max-width: 500px;
//     min-height: 220px;
//     background-color: lightgray;
//     grid-template-rows: 25% 75%;

//     > h1 {
//       background-color: grey;
//       width: 100%;
//       text-align: center;
//     }

//     > form {
//       display: flex;
//       margin: 10px;
//       flex-wrap: wrap;
//       align-items: center;
//       justify-content: center;
//       > div:first-child {
//         width: 100%;
//       }
//       > div {
//         display: flex;
//         justify-content: center;
//         align-items: center;
//         margin: 2px;

//         > input {
//           margin: 5px;
//         }

//         > label {
//           margin: 5px;
//         }
//       }

//       > .positions {
//         flex-wrap: wrap;

//         > .position {
//           display: grid;
//           grid-template-columns: 70% 15% 15%;
//           margin: 2px 0;

//           > input:last-child {
//             grid-column-start: 2;
//           }

//           > button {
//             padding: 0.5vh;
//           }
//         }
//       }

//       > .button-wrapper {
//         display: flex;
//         grid-column-start: 1;
//         grid-column-end: 3;
//         align-items: center;
//         justify-content: center;
//         width: 100%;

//         > button {
//           min-width: 55px;
//           padding: 5px 20px;
//         }
//       }
//     }
//   }
// }

.mdc-card-wrapper__text-section:first-child,
.mdc-card__media + .mdc-card-wrapper__text-section {
  padding-top: 16px;
}

.mdc-card-wrapper__text-section + .mdc-card-wrapper__text-section {
  padding-top: 18px;
}

.mdc-card-wrapper__text-section {
  padding-left: 16px;
  padding-right: 16px;
}

.list {
  // color: black;
  display: flex;
  gap: 10px;

  > .event {
    width: 335px;
    color: var(--mdc-theme-on-surface);

    .event__title {
      -moz-osx-font-smoothing: grayscale;
      -webkit-font-smoothing: antialiased;
      font-family: Roboto, sans-serif;
      font-family: var(
        --mdc-typography-headline6-font-family,
        var(--mdc-typography-font-family, Roboto, sans-serif)
      );
      font-size: 1.25rem;
      font-size: var(--mdc-typography-headline6-font-size, 1.25rem);
      line-height: 2rem;
      line-height: var(--mdc-typography-headline6-line-height, 2rem);
      font-weight: 500;
      font-weight: var(--mdc-typography-headline6-font-weight, 500);
      letter-spacing: 0.0125em;
      letter-spacing: var(--mdc-typography-headline6-letter-spacing, 0.0125em);
      // text-decoration: inherit;
      // -webkit-text-decoration: var(
      //   --mdc-typography-headline6-text-decoration,
      //   inherit
      // );
      // text-decoration: var(--mdc-typography-headline6-text-decoration, inherit);
      text-transform: inherit;
      text-transform: var(--mdc-typography-headline6-text-transform, inherit);
    }

    .event__subhead {
      -moz-osx-font-smoothing: grayscale;
      -webkit-font-smoothing: antialiased;
      font-family: Roboto, sans-serif;
      font-family: var(
        --mdc-typography-body2-font-family,
        var(--mdc-typography-font-family, Roboto, sans-serif)
      );
      font-size: 0.875rem;
      font-size: var(--mdc-typography-body2-font-size, 0.875rem);
      line-height: 1.25rem;
      line-height: var(--mdc-typography-body2-line-height, 1.25rem);
      font-weight: 400;
      font-weight: var(--mdc-typography-body2-font-weight, 400);
      letter-spacing: 0.0178571429em;
      letter-spacing: var(
        --mdc-typography-body2-letter-spacing,
        0.0178571429em
      );
      // text-decoration: inherit;
      // -webkit-text-decoration: var(
      //   --mdc-typography-body2-text-decoration,
      //   inherit
      // );
      // text-decoration: var(--mdc-typography-body2-text-decoration, inherit);
      text-transform: inherit;
      text-transform: var(--mdc-typography-body2-text-transform, inherit);
      opacity: 0.6;
    }

    .event__supporting-text {
      -moz-osx-font-smoothing: grayscale;
      -webkit-font-smoothing: antialiased;
      font-family: Roboto, sans-serif;
      font-family: var(
        --mdc-typography-body2-font-family,
        var(--mdc-typography-font-family, Roboto, sans-serif)
      );
      font-size: 0.875rem;
      font-size: var(--mdc-typography-body2-font-size, 0.875rem);
      line-height: 1.25rem;
      line-height: var(--mdc-typography-body2-line-height, 1.25rem);
      font-weight: 400;
      font-weight: var(--mdc-typography-body2-font-weight, 400);
      letter-spacing: 0.0178571429em;
      letter-spacing: var(
        --mdc-typography-body2-letter-spacing,
        0.0178571429em
      );
      text-decoration: inherit;
      -webkit-text-decoration: var(
        --mdc-typography-body2-text-decoration,
        inherit
      );
      text-decoration: var(--mdc-typography-body2-text-decoration, inherit);
      text-transform: inherit;
      text-transform: var(--mdc-typography-body2-text-transform, inherit);
      opacity: 0.6;
    }
  }

  // display: flex;
  // flex-wrap: wrap;
  // align-items: flex-start;
  // justify-content: left;

  // > .card {
  // flex: 0 0 20em;
  // min-height: 200px;
  // text-align: center;
  // width: 50vh;
  // display: grid;
  // align-content: center;
  // margin: 10px;
  // border-radius: 10px;
  // border-style: solid;
  // border-color: rgb(46, 46, 46);

  //   &.event {
  //     color: white;
  //   }

  //   > .cover {
  //     position: absolute;
  //     top: 0;
  //     left: 0;
  //     width: 100%;
  //     height: 100%;
  //     z-index: -1;
  //     opacity: 0.25;
  //     background-size: cover;
  //   }

  //   &.new {
  //     cursor: pointer;
  //   }

  //   > div {
  //     height: 100%;

  //     > span {
  //       font-size: 50px;
  //     }

  //     > .time {
  //       grid-row: 2 span 3;
  //     }
  //   }
  // }
}

.delete-icon-button {
  --mdc-theme-primary: #ff0000;
  --mdc-theme-on-primary: #2c2828;

  @include button.ink-color(#ffffff);
}

// .break {
//   flex-basis: 100%;
//   height: 0;
// }
</style>

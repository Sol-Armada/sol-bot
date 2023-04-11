<script setup>
import DeleteConfirm from "./DeleteConfirmComponent.vue";
import Edit from "./EditComponent.vue";
import { ref, onBeforeMount } from "vue";

const props = defineProps({
  event: Object,
});

const deleteRef = ref(null);
const editRef = ref(null);

const handleDelete = () => {
  deleteRef.value.openDeleteDialog(props.event._id);
};

const handleEdit = () => {
  editRef.value.show = true;
};

function openEvent(e) {
  const card = document.querySelector("#" + e);
  if (card.classList.contains("finished")) {
    console.log("true, something will happen here eventually");
  }
}

onBeforeMount(() => {
  const eventRef = ref(props.event);
  var startDate = new Date(eventRef.value.start);
  var endDate = new Date(eventRef.value.end);

  eventRef.value.start = startDate;
  eventRef.value.end = endDate;

  if (startDate.getDate() == endDate.getDate()) {
    eventRef.value._schedule =
      startDate.getMonth() +
      "/" +
      startDate.getDate() +
      "/" +
      startDate.getFullYear() +
      ", " +
      startDate.getHours() +
      ":" +
      startDate.getMinutes() +
      " - " +
      endDate.getHours() +
      ":" +
      endDate.getMinutes();
  } else {
    eventRef.value._schedule =
      startDate.getMonth() +
      "/" +
      startDate.getDate() +
      "/" +
      startDate.getFullYear() +
      ", " +
      startDate.getHours() +
      ":" +
      startDate.getMinutes() +
      " - " +
      endDate.getMonth() +
      "/" +
      endDate.getDate() +
      "/" +
      endDate.getFullYear() +
      ", " +
      endDate.getHours() +
      ":" +
      endDate.getMinutes();
  }
});
</script>
<template>
  <div
    class="event mdc-card"
    :key="event._id"
    :id="event._id"
    v-on:click="openEvent(event._id)"
  >
    <div
      class="my-card__media mdc-card__media mdc-card__media--16-9"
      :style="{ backgroundImage: 'url(' + event.cover + ')' }"
    ></div>
    <div class="mdc-card-wrapper__text-section">
      <div class="event__title">{{ event.name }}</div>
      <div class="event__subhead">{{ event._schedule }}</div>
    </div>
    <div class="description mdc-card-wrapper__text-section">
      <div class="event__supporting-text">
        {{ event.description }}
      </div>
    </div>
    <div class="mdc-card__actions" v-if="event.status <= 2">
      <button
        class="mdc-button mdc-button--leading mdc-card__action mdc-card__action--icon"
        title="Edit"
        v-on:click="handleEdit"
      >
        <span class="mdc-button__ripple"></span>
        <i class="material-icons mdc-button__icon" aria-hidden="true">edit</i>
        <span class="mdc-button__label">Edit</span>
      </button>
      <button
        class="delete-icon-button mdc-button mdc-button--raised mdc-button--leading mdc-card__action mdc-card__action--icon"
        title="Delete"
        v-on:click="handleDelete"
      >
        <span class="mdc-button__ripple"></span>
        <i class="material-icons mdc-button__icon" aria-hidden="true">
          delete
        </i>
        <span class="mdc-button__label">Delete</span>
      </button>
    </div>
    <DeleteConfirm ref="deleteRef" :event="event" />
    <Edit ref="editRef" :event="event" />
  </div>
</template>
<style lang="scss" scoped>
@use "@material/card";
@use "@material/button";
@import "../../assets/shadows.scss";

.event {
  width: 335px;
  color: var(--mdc-theme-on-surface);
  @include full_box_shadow(2, false);

  .description {
    flex-grow: 1;
    word-break: break-all;
  }

  .delete-icon-button {
    // --mdc-theme-primary: #ff0000;
    // --mdc-theme-on-primary: #2c2828;

    @include button.ink-color(#ffffff);
  }

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
    letter-spacing: var(--mdc-typography-body2-letter-spacing, 0.0178571429em);
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
    letter-spacing: var(--mdc-typography-body2-letter-spacing, 0.0178571429em);
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
</style>

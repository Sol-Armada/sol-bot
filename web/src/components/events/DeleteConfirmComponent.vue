<script setup>
import { onMounted } from "vue";
import { MDCDialog } from "@material/dialog";
import { deleteEvent } from "../../api";

const props = defineProps({
  event: Object,
});

onMounted(() => {
  const dialog = new MDCDialog(
    document.querySelector("#delete-" + props.event._id + ".mdc-dialog")
  );
  dialog.listen("MDCDialog:closing", (choice) => {
    if (choice.detail.action == "delete") {
      console.log(props.event._id);
      deleteEvent(props.event._id);
    }
  });
});

const openDeleteDialog = () => {
  const dialog = new MDCDialog(
    document.querySelector("#delete-" + props.event._id + ".mdc-dialog")
  );
  dialog.open();
};

defineExpose({ openDeleteDialog });
</script>
<template>
  <div class="mdc-dialog" :id="'delete-' + event._id">
    <div class="mdc-dialog__container">
      <div
        class="mdc-dialog__surface"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="my-dialog-title"
        aria-describedby="my-dialog-content"
      >
        <div class="mdc-dialog__content" id="my-dialog-content">
          Delete Event?
        </div>
        <div class="mdc-dialog__actions">
          <button
            type="button"
            class="mdc-button mdc-dialog__button"
            data-mdc-dialog-action="cancel"
          >
            <div class="mdc-button__ripple"></div>
            <span class="mdc-button__label">Cancel</span>
          </button>
          <button
            type="button"
            class="mdc-button mdc-dialog__button"
            data-mdc-dialog-action="delete"
          >
            <div class="mdc-button__ripple"></div>
            <span class="mdc-button__label">Delete</span>
          </button>
        </div>
      </div>
    </div>
    <div class="mdc-dialog__scrim"></div>
  </div>
</template>
<style lang="scss" scoped>
@use "@material/dialog";
</style>

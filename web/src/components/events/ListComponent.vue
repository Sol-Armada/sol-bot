<script setup>
import Card from "./CardComponent.vue";

const props = defineProps({
  admin: Object,
  events: Array,
});

function upcommingEvents() {
  if (props.events != undefined) {
    var e = [];
    props.events.forEach((ev) => {
      if (ev.status == 0) {
        e.push(ev);
      }
    });

    return e;
  }

  return [];
}

function finishedEvents() {
  if (props.events != undefined) {
    var e = [];
    props.events.forEach((ev) => {
      if (ev.status == 3) {
        e.push(ev);
      }
    });

    return e;
  }

  return [];
}

defineExpose({
  upcommingEvents,
  finishedEvents,
});
</script>

<template>
  <div class="list">
    <div class="cards">
      <span>
        <h1>Upcomming</h1>
      </span>
      <Card
        v-for="event in upcommingEvents()"
        :key="event._id"
        :event="event"
      />
      <div class="none" v-if="upcommingEvents().length == 0"><h2>None</h2></div>
    </div>

    <div class="cards">
      <span>
        <h1>Finished</h1>
      </span>
      <Card
        v-for="event in finishedEvents()"
        :key="event._id"
        :event="event"
        class="finished"
      />
      <div class="none" v-if="finishedEvents().length == 0"><h2>None</h2></div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@use "@material/fab";
@import "../../assets/shadows.scss";

.cards {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  gap: 10px;
  width: 100%;
  height: 100%;
  color: var(--mdc-theme-on-surface);
  background-color: var(--mdc-theme-surface);
  padding: 10px;
  margin-bottom: 10px;
  @include box_shadow(1, false);

  > .none {
    padding: 10px;
    text-align: center;
    flex-basis: 100%;
  }

  > span {
    width: 100%;
    text-transform: capitalize;
    position: sticky;
    padding: 5px 0;
    top: 0;
    background-color: var(--mdc-theme-surface);
    padding-left: 10px;
    border: 3px solid var(--mdc-theme-on-surface);
    border-radius: var(--mdc-shape-small, 4px) 10px var(--mdc-shape-small, 4px)
      10px;
    display: flex;
    align-items: center;
    z-index: 3;
    @include box_shadow(2, false);

    > h2 {
      position: absolute;
      right: 15px;
    }
  }

  .finished {
    cursor: pointer;
  }

  &:last-child {
    margin-bottom: 0;
  }
}
</style>

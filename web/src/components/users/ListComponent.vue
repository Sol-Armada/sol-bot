<script setup>
import { computed, ref, onMounted } from "vue";
import { updateUser } from "../../api";
import Card from "./CardComponent.vue";
import { MDCSwitch } from "@material/switch";

const props = defineProps({
  admin: Object,
  users: Array,
  updateUser: Function,
});

const sortBy = ref("name");
const searchFor = ref("");
const filterIssues = ref(false);

const admirals = computed(() => {
  if (props.users != undefined) {
    var users = filter(1, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const commanders = computed(() => {
  if (props.users != undefined) {
    var users = filter(2, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const lieutenants = computed(() => {
  if (props.users != undefined) {
    var users = filter(3, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const specialists = computed(() => {
  if (props.users != undefined) {
    var users = filter(4, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const technicians = computed(() => {
  if (props.users != undefined) {
    var users = filter(5, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const members = computed(() => {
  if (props.users != undefined) {
    var users = filter(6, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const recruits = computed(() => {
  if (props.users != undefined) {
    var users = filter(7, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const guests = computed(() => {
  if (props.users != undefined) {
    var users = filter(8, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const allies = computed(() => {
  if (props.users != undefined) {
    var users = filter(99, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});
const bots = computed(() => {
  if (props.users != undefined) {
    var users = filter(0, props.users);
    users = search(users);
    return sort(users);
  }

  return [];
});

function filter(rank, users) {
  users = users.filter((user) => user.rank == rank);

  if (filterIssues.value) {
    users = users.filter((user) => {
      if (user.bad_affiliation) {
        return user;
      }

      if (user.rank <= 6 && user.primary_org != "SOLARMADA") {
        return user;
      }

      if (!user.rsi_member) {
        return user;
      }
    });
  }

  return users;
}

function search(users) {
  if (searchFor.value == "") return users;
  var filteredUsers = [];
  users.forEach((user) => {
    if (user.name.toUpperCase().includes(searchFor.value.toUpperCase())) {
      filteredUsers.push(user);
    }
  });

  return filteredUsers;
}

function sort(users) {
  switch (sortBy.value) {
    case "name":
      users = users.sort((a, b) => {
        if (a.name < b.name) {
          return -1;
        } else if (a.name > b.name) {
          return 1;
        } else {
          return 0;
        }
      });
      break;
    case "events":
      users = users.sort((a, b) => {
        if (a.events > b.events) {
          return -1;
        } else if (a.events < b.events) {
          return 1;
        } else {
          return 0;
        }
      });
      break;
  }

  return users;
}

onMounted(() => {
  for (const el of document.querySelectorAll(".mdc-switch")) {
    // eslint-disable-next-line no-unused-vars
    const switchControl = new MDCSwitch(el);
  }
});
</script>

<template>
  <form onsubmit="event.preventDefault();" role="search">
    <input
      id="search"
      type="search"
      placeholder="Search..."
      autofocus
      v-model="searchFor"
    />
    <select name="sort" id="sort" v-model="sortBy">
      <option value="name">Sort by: Name</option>
      <option value="events">Sort by: Events</option>
    </select>
    <div class="switch">
      <button
        class="mdc-switch mdc-switch--unselected"
        type="button"
        role="switch"
        aria-checked="false"
        v-on:click="filterIssues = !filterIssues"
      >
        <div class="mdc-switch__track"></div>
        <div class="mdc-switch__handle-track">
          <div class="mdc-switch__handle">
            <div class="mdc-switch__shadow">
              <div class="mdc-elevation-overlay"></div>
            </div>
            <div class="mdc-switch__ripple"></div>
            <div class="mdc-switch__icons">
              <svg
                class="mdc-switch__icon mdc-switch__icon--on"
                viewBox="0 0 24 24"
              >
                <path
                  d="M19.69,5.23L8.96,15.96l-4.23-4.23L2.96,13.5l6,6L21.46,7L19.69,5.23z"
                />
              </svg>
              <svg
                class="mdc-switch__icon mdc-switch__icon--off"
                viewBox="0 0 24 24"
              >
                <path d="M20 13H4v-2h16v2z" />
              </svg>
            </div>
          </div>
        </div>
      </button>
      <label for="basic-switch">With Issues</label>
    </div>
  </form>
  <div class="list">
    <div class="cards" v-if="admirals.length > 0">
      <span style="border-color: #235cff"><h1>admirals</h1></span>
      <Card :users="admirals" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="commanders.length > 0">
      <span style="border-color: #fff"><h1>commanders</h1></span>
      <Card :users="commanders" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="lieutenants.length > 0">
      <span style="border-color: #5796ff"><h1>lieutenants</h1></span>
      <Card :users="lieutenants" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="specialists.length > 0">
      <span style="border-color: #da5c5c"><h1>specialists</h1></span>
      <Card :users="specialists" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="technicians.length > 0">
      <span style="border-color: #e69737"><h1>technicians</h1></span>
      <Card :users="technicians" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="members.length > 0">
      <span style="border-color: #ffc900"><h1>members</h1></span>
      <Card :users="members" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="recruits.length > 0">
      <span style="border-color: #1cfac0"><h1>recruits</h1></span>
      <Card :users="recruits" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="guests.length > 0">
      <span><h1>guests</h1></span>
      <Card :users="guests" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="allies.length > 0">
      <span style="border-color: #e05b03"><h1>allies</h1></span>
      <Card :users="allies" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="bots.length > 0">
      <span><h1>bots</h1></span>
      <Card :users="bots" :updateUser="updateUser"></Card>
    </div>
    <div class="nothing-found">
      <h1>Nothing Found</h1>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@use "@material/switch/styles";
@use "@material/switch";
@import "../../assets/shadows.scss";

form {
  display: flex;
  justify-content: left;
  align-items: center;
  margin: 10px 0;

  > * {
    @include box_shadow(1, false);
    color: var(--mdc-theme-on-surface);
    background-color: var(--mdc-theme-surface);
    border-radius: var(--mdc-shape-small, 4px);
    border: 0;
    z-index: 1;
  }

  > input[type="search"] {
    height: 3rem;
    font-size: 1.8rem;
    padding: 0 1.6rem;
    appearance: none;
    position: relative;
    outline: none !important;
    margin-right: 10px;
  }

  > select[name="sort"] {
    height: 2rem;
    font-size: 1rem;
    padding: 0 1rem;
    appearance: none;
    position: relative;
    outline: none !important;
    margin-right: 10px;
  }

  > .switch {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 2rem;
    padding: 0 1rem;

    > .mdc-switch {
      @include switch.theme(
        (
          handle-surface-color: var(--mdc-theme-primary),
          selected-handle-color: var(--mdc-theme-primary),
          selected-hover-handle-color: var(--mdc-theme-primary),
          selected-hover-track-color: var(--mdc-theme-primary),
          selected-track-color: var(--mdc-theme-primary),
          selected-hover-state-layer-color: var(--mdc-theme-primary),
          selected-focus-handle-color: var(--mdc-theme-primary),
          selected-focus-track-color: var(--mdc-theme-primary),
          selected-pressed-handle-color: var(--mdc-theme-primary),
          selected-pressed-track-color: var(--mdc-theme-primary),

          unselected-pressed-handle-color: var(--mdc-theme-primary),
          unselected-pressed-track-color: var(--mdc-theme-primary),
        )
      );
      margin-right: 5px;
    }
  }
}

.cards {
  // display: flex;
  // gap: 10px;
  // flex-wrap: wrap;
  color: var(--mdc-theme-on-surface);
  background-color: var(--mdc-theme-surface);
  padding: 10px;
  margin-bottom: 10px;
  @include box_shadow(1, false);

  > span {
    width: 100%;
    text-transform: capitalize;
    position: sticky;
    padding: 5px 0;
    top: 0;
    background-color: var(--mdc-theme-surface);
    padding-left: 10px;
    border: 3px solid var(--mdc-theme-on-surface);
    // border-left: 3px solid var(--mdc-theme-on-surface);
    border-radius: var(--mdc-shape-small, 4px) 10px var(--mdc-shape-small, 4px) 10px;
    display: flex;
    align-items: center;
    z-index: 10;
    @include box_shadow(2, false);
  }
  &:last-child {
    margin-bottom: 0;
  }
}

// input {
//   height: 3rem;
//   border: 0;
//   color: var(--color-text);
//   font-size: 1.8rem;
//   margin-bottom: 10px;
// }

input[type="number"] {
  width: 30%;
}

.nothing-found {
  display: none;
  width: 100%;
  min-height: 300px;
  justify-content: center;
  align-items: center;

  &:only-child {
    display: flex;
  }
}

// button {
//   padding: 0.5rem;
//   margin: 0 0.25rem;
//   border-radius: 0.5rem;
//   border-color: var(--color-border);
//   background: var(--color-background-soft);
//   color: var(--color-text);

//   &:hover {
//     border-color: var(--color-border-hover);
//     background-color: var(--color-background);
//   }
// }

.hidden {
  display: none;
}
</style>

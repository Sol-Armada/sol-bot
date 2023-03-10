<script setup>
import { computed, ref } from "vue";
import { updateUser } from "../../api";
import Card from "./CardComponent.vue";

const props = defineProps({
  admin: Object,
  users: Array,
  updateUser: Function,
});

const sortBy = ref("name");
const searchFor = ref("");

const bots = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 0);
    users = search(users);
    return sort(users);
  }

  return [];
});
const admirals = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 1);
    users = search(users);
    return sort(users);
  }

  return [];
});
const commanders = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 2);
    users = search(users);
    return sort(users);
  }

  return [];
});
const lieutenants = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 3);
    users = search(users);
    return sort(users);
  }

  return [];
});
const specialists = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 4);
    users = search(users);
    return sort(users);
  }

  return [];
});
const technicians = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 5);
    users = search(users);
    return sort(users);
  }

  return [];
});
const members = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 6);
    users = search(users);
    return sort(users);
  }

  return [];
});
const recruits = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 7);
    users = search(users);
    return sort(users);
  }

  return [];
});
const guests = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 8);
    users = search(users);
    return sort(users);
  }

  return [];
});
const allies = computed(() => {
  if (props.users != undefined) {
    var users = props.users.filter((u) => u.rank == 99);
    users = search(users);
    return sort(users);
  }

  return [];
});

// var delayTimer;
function search(users) {
  if (searchFor.value == "") return users;
  var filteredUsers = [];
  users.forEach((user) => {
    if (user.name.toUpperCase().includes(searchFor.value.toUpperCase())) {
      filteredUsers.push(user);
    }
  });

  return filteredUsers;
  // clearTimeout(delayTimer);
  // delayTimer = setTimeout(() => {
  //   const cardLists = document.querySelectorAll(".cards");
  //   cardLists.forEach((cl) => {
  //     cl.classList.add("hidden");
  //     const cards = cl.querySelectorAll(".card");
  //     cards.forEach((c) => {
  //       if (by != "") {
  //         if (c.dataset.nick.toUpperCase().includes(by.toUpperCase())) {
  //           c.classList.remove("hidden");
  //         } else {
  //           c.classList.add("hidden");
  //         }
  //       } else {
  //         c.classList.remove("hidden");
  //       }
  //       if (!c.classList.contains("hidden")) {
  //         cl.classList.remove("hidden");
  //       }
  //     });
  //   });
  // }, 250);
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
    <div>
      <select name="sort" id="sort" v-model="sortBy">
        <option value="name">Sort by: Name</option>
        <option value="events">Sort by: Events</option>
      </select>
    </div>
  </form>
  <div class="list">
    <div class="cards" v-if="admirals.length > 0">
      <h1>admirals</h1>
      <Card :users="admirals" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="commanders.length > 0">
      <h1>
        commanders
        <hr />
      </h1>
      <Card :users="commanders" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="lieutenants.length > 0">
      <h1>
        lieutenants
        <hr />
      </h1>
      <Card :users="lieutenants" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="specialists.length > 0">
      <h1>
        specialists
        <hr />
      </h1>
      <Card :users="specialists" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="technicians.length > 0">
      <h1>
        technicians
        <hr />
      </h1>
      <Card :users="technicians" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="members.length > 0">
      <h1>
        members
        <hr />
      </h1>
      <Card :users="members" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="recruits.length > 0">
      <h1>
        recruits
        <hr />
      </h1>
      <Card :users="recruits" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="guests.length > 0">
      <h1>
        guests
        <hr />
      </h1>
      <Card :users="guests" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="allies.length > 0">
      <h1>
        allies
        <hr />
      </h1>
      <Card :users="allies" :updateUser="updateUser"></Card>
    </div>
    <div class="cards" v-if="bots.length > 0">
      <h1>
        bots
        <hr />
      </h1>
      <Card :users="bots" :updateUser="updateUser"></Card>
    </div>
    <div class="nothing-found">
      <h1>Nothing Found</h1>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@import "../../assets/shadows.scss";

form {
  display: flex;
  justify-content: center;
  align-items: center;
}

.cards {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  color: var(--mdc-theme-on-surface);
  background-color: var(--mdc-theme-surface);
  padding: 0 10px;
  padding-bottom: 10px;
  margin-bottom: 10px;
  @include box_shadow(1, false);

  > h1 {
    width: 100%;
    text-transform: capitalize;
    margin: 1rem 0;
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

input[type="search"] {
  height: 3rem;
  border: 0;
  border-radius: var(--mdc-shape-small, 4px);
  color: var(--mdc-theme-on-surface);
  font-size: 1.8rem;
  margin-bottom: 10px;
  background: var(--mdc-theme-surface);
  padding: 0 1.6rem;
  appearance: none;
  z-index: 1;
  position: relative;
  outline: none !important;
  margin-right: 10px;

  @include box_shadow(1, false);
}

select[name="sort"] {
  height: 2rem;
  border: 0;
  border-radius: var(--mdc-shape-small, 4px);
  color: var(--mdc-theme-on-surface);
  font-size: 1rem;
  margin-bottom: 10px;
  background: var(--mdc-theme-surface);
  padding: 0 1rem;
  appearance: none;
  z-index: 1;
  position: relative;
  outline: none !important;

  @include box_shadow(1, false);
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

button {
  padding: 0.5rem;
  margin: 0 0.25rem;
  border-radius: 0.5rem;
  border-color: var(--color-border);
  background: var(--color-background-soft);
  color: var(--color-text);

  &:hover {
    border-color: var(--color-border-hover);
    background-color: var(--color-background);
  }
}
</style>

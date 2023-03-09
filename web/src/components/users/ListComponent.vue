<script setup>
import { computed } from "vue";
import { updateUser } from "../../api";
import { truncateString } from "../../utils";
import Card from "./CardComponent.vue";

const Ranks = {
  0: { name: "Bot" },
  1: { name: "Admiral" },
  2: { name: "Commander" },
  3: { name: "Lieutenant" },
  4: { name: "Specialist" },
  5: { name: "Technician" },
  6: { name: "Member" },
  7: { name: "Recruit" },
  8: { name: "Guest" },
  99: { name: "Ally" },
};

const props = defineProps({
  admin: Object,
  users: Array,
  updateUser: Function,
});
const bots = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 0);
  }

  return [];
});
const admirals = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 1);
  }

  return [];
});
const commanders = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 2);
  }

  return [];
});
const lieutenants = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 3);
  }

  return [];
});
const specialists = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 4);
  }

  return [];
});
const technicians = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 5);
  }

  return [];
});
const members = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 6);
  }

  return [];
});
const recruits = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 7);
  }

  return [];
});
const guests = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 8);
  }

  return [];
});
const allies = computed(() => {
  if (props.users != undefined) {
    return props.users.filter((u) => u.rank == 99);
  }

  return [];
});

var delayTimer;
function search(e) {
  var value = e.srcElement.value.toUpperCase();
  clearTimeout(delayTimer);
  delayTimer = setTimeout(() => {
    const cardLists = document.querySelectorAll(".cards");
    cardLists.forEach((cl) => {
      cl.classList.add("hidden");
      const cards = cl.querySelectorAll(".card");
      cards.forEach((c) => {
        if (value != "") {
          if (c.dataset.nick.toUpperCase().includes(value)) {
            c.classList.remove("hidden");
          } else {
            c.classList.add("hidden");
          }
        } else {
          c.classList.remove("hidden");
        }
        if (!c.classList.contains("hidden")) {
          cl.classList.remove("hidden");
        }
      });
    });
  }, 250);
}
</script>

<template>
  <form onsubmit="event.preventDefault();" role="search">
    <input
      id="search"
      type="search"
      placeholder="Search..."
      autofocus
      v-on:keyup="search"
    />
  </form>
  <div class="cards" v-if="admirals.length > 0">
    <h1>
      admirals
      <hr />
    </h1>
    <Card :users="admirals" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in admirals"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          user.primary_org == 'SOLARMADA'
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
  <div class="cards" v-if="commanders.length > 0">
    <h1>
      commanders
      <hr />
    </h1>
    <Card :users="commanders" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in commanders"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          user.primary_org == 'SOLARMADA'
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
  <div class="cards" v-if="lieutenants.length > 0">
    <h1>
      lieutenants
      <hr />
    </h1>
    <Card :users="lieutenants" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in lieutenants"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          user.primary_org == 'SOLARMADA'
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
  <div class="cards" v-if="specialists.length > 0">
    <h1>
      specialists
      <hr />
    </h1>
    <Card :users="specialists" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in specialists"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          user.primary_org == 'SOLARMADA'
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
  <div class="cards" v-if="technicians.length > 0">
    <h1>
      technicians
      <hr />
    </h1>
    <Card :users="technicians" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in technicians"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          user.primary_org == 'SOLARMADA'
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
  <div class="cards" v-if="members.length > 0">
    <h1>
      members
      <hr />
    </h1>
    <Card :users="members" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in members"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          user.primary_org == 'SOLARMADA'
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
  <div class="cards" v-if="recruits.length > 0">
    <h1>
      recruits
      <hr />
    </h1>
    <Card :users="recruits" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in recruits"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>
      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
  <div class="cards" v-if="guests.length > 0">
    <h1>
      guests
      <hr />
    </h1>
    <Card :users="guests" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in guests"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
    </div> -->
  </div>
  <div class="cards" v-if="allies.length > 0">
    <h1>
      allies
      <hr />
    </h1>
    <Card :users="allies" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in allies"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          user.primary_org == 'SOLARMADA'
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
  <div class="cards" v-if="bots.length > 0">
    <h1>
      bots
      <hr />
    </h1>
    <Card :users="bots" :updateUser="updateUser"></Card>
    <!-- <div
      v-for="user in bots"
      :key="user.id"
      :class="
        'card ' +
        ((user.primary_org == 'SOLARMADA' ||
          user.primary_org == '' ||
          user.rank == 0 ||
          user.rank >= 6 ||
          user.rank == 99) &&
        user.primary_org != 'REDACTED' &&
        user.bad_affiliation != true
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org ')
      "
      :id="user.id"
      :data-nick="user.name"
    >
      <h2>
        {{ truncateString(user.name, 14) }}
        <hr />
      </h2>

      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.primary_org != 'REDACTED'
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        REDACTED ORG
      </h3>
      <h3 v-if="user.bad_affiliation == true">BAD ORG</h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="
          user.rank != 0 &&
          user.rank <= 8 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          user.primary_org == 'SOLARMADA'
        "
      >
        <h3>Events</h3>
        <div>
          <button
            class="material-symbols-outlined"
            v-on:click="
              user.events--;
              updateUser(user);
            "
          >
            remove
          </button>
          <span class="count">{{ user.events }}</span>
          <button
            class="material-symbols-outlined"
            v-if="Ranks[user.rank - 1]"
            v-on:click="
              user.events++;
              updateUser(user);
            "
          >
            add
          </button>
        </div>
      </div>
    </div> -->
  </div>
</template>

<style lang="scss" scoped>
@import "../../assets/shadows.scss";

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

  @include box_shadow(1, false);
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

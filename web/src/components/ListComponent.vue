<script setup>
import { computed } from "vue";
import { updateUser } from "../api";
import { truncateString } from "../utils";

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
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="commanders.length > 0">
    <h1>
      commanders
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="lieutenants.length > 0">
    <h1>
      lieutenants
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="specialists.length > 0">
    <h1>
      specialists
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="technicians.length > 0">
    <h1>
      technicians
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="members.length > 0">
    <h1>
      members
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="recruits.length > 0">
    <h1>
      recruits
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="guests.length > 0">
    <h1>
      guests
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="allies.length > 0">
    <h1>
      allies
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
  <div class="cards" v-if="bots.length > 0">
    <h1>
      bots
      <hr />
    </h1>
    <div
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
          user.primary_org != 'REDACTED' &&
          user.rank <= 6
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
    </div>
  </div>
</template>

<style lang="scss">
.cards {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;

  > h1 {
    width: 100%;
    text-transform: capitalize;
    margin: 1rem 0;
  }

  .card {
    opacity: 1;
    text-align: center;
    border: 2px solid grey;
    border-radius: 10px;
    width: 200px;
    height: 200px;

    hr {
      width: 80%;
      margin: auto;
    }

    .events {
      display: flex;
      flex-direction: column;

      div {
        display: flex;
        justify-content: center;
        align-items: center;

        button {
          background: transparent;
          border: 2px solid var(--color-border);
          border-radius: 10px;
          padding: 4px;
          margin: 0 10px;
          cursor: pointer;
          color: var(--color-text);

          &:nth-child(odd) {
            cursor: pointer;
          }
        }
      }
    }

    .controls {
      position: absolute;
      width: 100%;
      bottom: 0;
      padding: 5px;

      .promote,
      .demote,
      .ally {
        background: transparent;
        border: 2px solid var(--color-border);
        border-radius: 10px;
        padding: 10px;
        cursor: pointer;
        color: var(--color-text);
      }

      .demote:not(:only-child) {
        display: none;
      }
    }

    &.ally {
      border-color: #e05b03;

      hr {
        border-color: #e05b03;
      }
    }

    &.bad-org {
      border-color: #e00303;
      box-shadow: inset 0 0 30px 1px #ff0000;

      hr {
        border-color: #e00303;
      }
    }

    &.recruit {
      border-color: #ffffff;

      hr {
        border-color: #ffffff;
      }
    }

    &.recruit {
      border-color: #1cfac0;

      hr {
        border-color: #1cfac0;
      }
    }

    &.member {
      border-color: #ffc900;

      hr {
        border-color: #ffc900;
      }
    }

    &.specialist {
      border-color: #da5c5c;

      hr {
        border-color: #da5c5c;
      }
    }

    &.technician {
      border-color: #e69737;

      hr {
        border-color: #e69737;
      }
    }

    &.lieutenant {
      border-color: #5796ff;

      hr {
        border-color: #5796ff;
      }
    }

    &.commander {
      border-color: white;

      hr {
        border-color: white;
      }
    }

    &.admiral {
      border-color: #235cff;

      hr {
        border-color: #235cff;
      }
    }
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
  color: var(--color-text);
  font-size: 1.8rem;
  margin-bottom: 10px;
  background: var(--color-background-soft);
  padding: 0 1.6rem;
  border-radius: 0.7rem;
  appearance: none;
  z-index: 1;
  position: relative;
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

.hidden {
  display: none;
}
</style>

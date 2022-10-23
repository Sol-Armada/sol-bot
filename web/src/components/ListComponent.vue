<script setup>
import { updateUser } from "../api";
import { truncateString } from "../utils";

defineProps({
  admin: Object,
  users: Array,
  updateUser: Function,
});

const Ranks = {
  0: { name: "Bot", minEvents: 0 },
  1: { name: "Admiral", minEvents: 0 },
  2: { name: "Commander", minEvents: 0 },
  3: { name: "Lieutenant", minEvents: 0 },
  4: { name: "Specialist", minEvents: 20 },
  5: { name: "Technician", minEvents: 10 },
  6: { name: "Member", minEvents: 3 },
  7: { name: "Recruit", minEvents: 0 },
  99: { name: "Ally", minEvents: 0 },
};

var delayTimer;
function search(e) {
  var value = e.srcElement.value.toUpperCase();
  clearTimeout(delayTimer);
  delayTimer = setTimeout(() => {
    const cards = document.querySelectorAll(".card");
    console.log(cards);
    if (value != "") {
      cards.forEach((card) => {
        if (card.dataset.nick.toUpperCase().includes(value)) {
          card.classList.remove("hidden");
        } else {
          card.classList.add("hidden");
        }
      });
    } else {
      cards.forEach((card) => {
        card.classList.remove("hidden");
      });
    }
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
  <div class="cards">
    <div
      v-for="user in users"
      :key="user.id"
      :class="
        'card ' +
        (user.primary_org == 'SOLARMADA' ||
        user.primary_org == '' ||
        user.rank == 0 ||
        user.rank >= 6 ||
        user.rank == 99
          ? Ranks[user.rank].name.toLowerCase()
          : 'bad-org')
      "
      :id="user.id"
      :data-nick="user.nick"
    >
      <h2>{{ truncateString(user.nick, 14) }}</h2>
      <hr />
      <h3>{{ Ranks[user.rank].name }}</h3>
      <hr />
      <h3
        v-if="
          user.primary_org != '' &&
          user.primary_org != 'SOLARMADA' &&
          user.rank <= 6
        "
      >
        <a
          :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
          target="_blank"
          >{{ user.primary_org }}</a
        >
      </h3>
      <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
        Not on RSI
      </h3>
      <div
        class="events"
        v-if="user.rsi_member && user.rank != 0 && user.rank != 99"
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
      <div
        class="controls"
        v-if="
          false &&
          user.rank != 0 &&
          user.rank != 99 &&
          admin.rank < user.rank &&
          admin.id != user.id
        "
      >
        <button
          class="promote"
          v-if="
            Ranks[user.rank - 1] &&
            user.rank - 1 != 0 &&
            user.events >= Ranks[user.rank - 1].minEvents &&
            admin.rank >= 3
          "
          v-on:click="
            user.rank--;
            updateUser(user);
          "
        >
          Promote
        </button>
        <button
          class="demote"
          v-if="user.rank != 0 && Ranks[user.rank + 1]"
          v-on:click="
            user.rank++;
            updateUser(user);
          "
        >
          Demote
        </button>
        <button
          class="ally"
          v-if="user.rank == 7"
          v-on:click="
            user.ally = true;
            user.rank = 99;
            updateUser(user);
          "
        >
          Ally
        </button>
      </div>
    </div>
  </div>
</template>

<style lang="scss">
.cards {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;

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

    &.hidden {
      display: none;
      opacity: 0;
      transition: opacity 1s ease;
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
</style>

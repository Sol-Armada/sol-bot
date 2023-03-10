<script setup>
import { truncateString, getRankName } from "../../utils";

defineProps({
  users: Array,
  updateUser: Function,
});
</script>
<template>
  <div
    v-for="user in users"
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
        ? getRankName(user.rank).toLowerCase()
        : 'bad-org ')
    "
    :id="user.id"
    :data-nick="user.name"
  >
    <h2>
      {{ truncateString(user.name, 14) }}
      <hr />
    </h2>

    <div
      v-if="
        user.primary_org != '' &&
        user.primary_org != 'SOLARMADA' &&
        user.primary_org != 'REDACTED' &&
        user.rank <= 7
      "
    >
      <h3>Different Primary</h3>
      <a
        :href="'https://robertsspaceindustries.com/orgs/' + user.primary_org"
        target="_blank"
        class="other-org mdc-button mdc-button--raised mdc-button--icon-trailing"
      >
        <span class="mdc-button__label">{{ user.primary_org }}</span>
        <i class="material-icons mdc-button__icon" aria-hidden="true"
          >open_in_new</i
        >
      </a>
    </div>
    <h3 v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false">
      REDACTED ORG
    </h3>
    <div v-if="user.bad_affiliation == true">
      <h3>ENEMY ORG</h3>
      <a
        :href="'https://robertsspaceindustries.com/citizens/' + user.name + '/organizations'"
        target="_blank"
        class="other-org mdc-button mdc-button--raised mdc-button--icon-trailing"
      >
        <span class="mdc-button__label">affiliates</span>
        <i class="material-icons mdc-button__icon" aria-hidden="true"
          >open_in_new</i
        >
      </a>
    </div>
    <h3 v-if="!user.rsi_member && user.rank != 0 && user.rank != 99">
      Not on RSI
    </h3>
    <div
      class="events"
      v-if="
        user.rank > 0 &&
        user.rank <= 7 &&
        user.bad_affiliation == false &&
        user.primary_org != 'REDACTED' &&
        user.rsi_member == true &&
        (user.primary_org == 'SOLARMADA' || user.rank == 7)
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
</template>
<style lang="scss" scoped>
@use "@material/card";
@use "@material/button";

@import "../../assets/shadows.scss";

.card {
  opacity: 1;
  text-align: center;
  @include full_box_shadow(2, false);
  width: 200px;
  height: 200px;
  color: var(--mdc-theme-on-surface);
  background-color: var(--mdc-theme-surface);

  hr {
    width: 80%;
    margin: auto;
  }

  h3 {
    margin-top: 5px;
  }

  .events {
    display: flex;
    flex-direction: column;

    div {
      display: flex;
      justify-content: center;
      align-items: center;
      margin: 10px 0;

      button {
        background: transparent;
        // border: 2px solid var(--color-border);
        border-radius: 5px;
        padding: 4px;
        margin: 0 10px;
        cursor: pointer;

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
      border: 2px solid transparent;
      border-radius: 10px;
      padding: 10px;
      cursor: pointer;
    }

    .demote:not(:only-child) {
      display: none;
    }
  }

  a.other-org {
    margin-top: 10px;
  }

  &.ally {
    // border-color: #e05b03;

    hr {
      border-color: #e05b03;
    }
  }

  &.bad-org {
    // border-color: #e00303;
    // box-shadow: inset 0 0 30px 1px #ff0000;

    hr {
      border-color: #e00303;
    }
  }

  &.recruit {
    // border-color: #1cfac0;

    hr {
      border-color: #1cfac0;
    }
  }

  &.member {
    // border-color: #ffc900;

    hr {
      border-color: #ffc900;
    }
  }

  &.specialist {
    // border-color: #da5c5c;

    hr {
      border-color: #da5c5c;
    }
  }

  &.technician {
    // border-color: #e69737;

    hr {
      border-color: #e69737;
    }
  }

  &.lieutenant {
    // border-color: #5796ff;

    hr {
      border-color: #5796ff;
    }
  }

  &.commander {
    // border-color: white;

    hr {
      border-color: white;
    }
  }

  &.admiral {
    // border-color: #235cff;

    hr {
      border-color: #235cff;
    }
  }
}
</style>

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
    <h2>{{ truncateString(user.name, 14) }}</h2>

    <div class="events">
      <div
        class="bad-primary"
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
          <i class="material-icons mdc-button__icon" aria-hidden="true">
            open_in_new
          </i>
        </a>
      </div>

      <div
        class="redacted-org"
        v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false"
      >
        <h3>REDACTED ORG</h3>
      </div>

      <div class="enemy-org" v-if="user.bad_affiliation == true">
        <h3>ENEMY ORG</h3>
        <a
          :href="
            'https://robertsspaceindustries.com/citizens/' +
            user.name +
            '/organizations'
          "
          target="_blank"
          class="other-org mdc-button mdc-button--raised mdc-button--icon-trailing"
        >
          <span class="mdc-button__label">affiliates</span>
          <i class="material-icons mdc-button__icon" aria-hidden="true">
            open_in_new
          </i>
        </a>
      </div>

      <div
        class="not-on-rsi"
        v-if="!user.rsi_member && user.rank != 0 && user.rank != 99"
      >
        <h3>Not on RSI</h3>
      </div>

      <div
        class="controls"
        v-if="
          user.rank > 0 &&
          user.rank <= 7 &&
          user.bad_affiliation == false &&
          user.primary_org != 'REDACTED' &&
          user.rsi_member == true &&
          (user.primary_org == 'SOLARMADA' || user.rank == 7)
        "
      >
        <h3>Event Count</h3>
        <button
          v-on:click="
            user.events--;
            updateUser(user);
          "
        >
          <i class="material-icons" aria-hidden="true">remove</i>
        </button>
        <span class="count">{{ user.events }}</span>
        <button
          v-on:click="
            user.events++;
            updateUser(user);
          "
        >
          <i class="material-icons" aria-hidden="true">add</i>
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
  display: flex;
  // text-align: center;
  @include full_box_shadow(2, false);
  width: 100%;
  height: 50px;
  color: var(--mdc-theme-on-surface);
  background-color: var(--mdc-theme-surface);
  margin: 5px 0;
  align-items: center;

  > h2 {
    margin-left: 10px;
  }

  > .events {
    display: flex;
    position: absolute;
    right: 0;

    > *:not(:last-child) {
      margin-right: 10px;
    }

    div {
      display: flex;
      justify-content: center;
      align-items: center;
      margin: 10px;

      > h3 {
        margin-right: 10px;
      }

      &.controls {
        button {
          background-color: var(--mdc-theme-primary);
          border-radius: var(--mdc-shape-small, 4px);
          border-style: none;
          margin: 10px;

          @include box_shadow(1, false);
        }

        span {
          width: 25px;
          text-align: center;
        }
      }

      // button {
      //   background: transparent;
      //   // border: 2px solid var(--color-border);
      //   border-radius: 5px;
      //   padding: 4px;
      //   // margin: 0 10px;
      //   cursor: pointer;

      //   &:nth-child(odd) {
      //     cursor: pointer;
      //   }
      // }
    }
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

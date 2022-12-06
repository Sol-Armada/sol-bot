<script setup>
defineProps({
  user: Object,
  updateUser: Function,
});
</script>
<template>
  <div
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
    <h3 v-if="user.primary_org == 'REDACTED' && user.bad_affiliation == false">
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
</template>

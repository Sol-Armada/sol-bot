<script setup>
import { ref, computed, onMounted } from "vue";
import { useComposition } from "@/compositions";
import { filterForRank } from "../utils";
import { getUsers, getBankBalance } from "../api";

const { users, bank } = useComposition();

const usersRef = ref(users);

const admiralCount = computed(() => {
  if (usersRef.value != undefined) {
    var users = filterForRank(1, usersRef.value);
    return users.length;
  }

  return 0;
});
const commanderCount = computed(() => {
  if (usersRef.value != undefined) {
    var users = filterForRank(2, usersRef.value);
    return users.length;
  }

  return 0;
});
const lieutenantCount = computed(() => {
  if (usersRef.value != undefined) {
    var users = filterForRank(3, usersRef.value);
    return users.length;
  }

  return 0;
});
const specialistCount = computed(() => {
  if (usersRef.value != undefined) {
    var users = filterForRank(4, usersRef.value);
    return users.length;
  }

  return 0;
});
const technicianCount = computed(() => {
  if (usersRef.value != undefined) {
    var users = filterForRank(5, usersRef.value);
    return users.length;
  }

  return 0;
});
const memberCount = computed(() => {
  if (usersRef.value != undefined) {
    var users = filterForRank(6, usersRef.value);
    return users.length;
  }

  return 0;
});
const recruitCount = computed(() => {
  if (usersRef.value != undefined) {
    var users = filterForRank(7, usersRef.value);
    return users.length;
  }

  return 0;
});
</script>

<template>
  <div class="dashboard">
    <div class="card admiral">
      <div>
        <h1 class="title">Admirals</h1>
        <hr />
        <div class="content">
          <h2>{{ admiralCount }}</h2>
        </div>
      </div>
    </div>
    <div class="card commander">
      <div>
        <h1 class="title">Commanders</h1>
        <hr />
        <div class="content">
          <h2>{{ commanderCount }}</h2>
        </div>
      </div>
    </div>
    <div class="card lieutenant">
      <div>
        <h1 class="title">Lieutenants</h1>
        <hr />
        <div class="content">
          <h2>{{ lieutenantCount }}</h2>
        </div>
      </div>
    </div>
    <div class="card specialist">
      <div>
        <h1 class="title">Specialists</h1>
        <hr />
        <div class="content">
          <h2>{{ specialistCount }}</h2>
        </div>
      </div>
    </div>
    <div class="card technician">
      <div>
        <h1 class="title">Technicians</h1>
        <hr />
        <div class="content">
          <h2>{{ technicianCount }}</h2>
        </div>
      </div>
    </div>
    <div class="card member">
      <div>
        <h1 class="title">Members</h1>
        <hr />
        <div class="content">
          <h2>{{ memberCount }}</h2>
        </div>
      </div>
    </div>
    <div class="card recruit">
      <div>
        <h1 class="title">Recruits</h1>
        <hr />
        <div class="content">
          <h2>{{ recruitCount }}</h2>
        </div>
      </div>
    </div>
    <div class="break"></div>
    <div class="card">
      <div>
        <h1 class="title">Bank Balance</h1>
        <hr />
        <div class="content">
          <h2>{{ bank.balance }}</h2>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@import "../assets/shadows.scss";
@import "../assets/variables.scss";

.dashboard {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  color: var(--mdc-theme-on-surface);
  width: 100%;
  padding: 10px 0;

  .card {
    display: flex;
    background-color: var(--mdc-theme-surface);
    padding: 10px;
    gap: 10px;

    @include box_shadow(1, false);

    &.admiral .title {
      color: $admiral;
    }

    &.commander .title {
      color: $commander;
    }

    &.lieutenant .title {
      color: $lieutenant;
    }

    &.specialist .title {
      color: $specialist;
    }

    &.technician .title {
      color: $technician;
    }

    &.member .title {
      color: $member;
    }

    &.recruit .title {
      color: $recruit;
    }
    .content {
      padding-top: 10px;
      text-align: center;
    }
  }
}
</style>

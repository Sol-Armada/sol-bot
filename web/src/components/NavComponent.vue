<script setup>
import { useComposition } from "../compositions";
import { onMounted } from "vue";
import { MDCList } from "@material/list";
import { getRankName, averageColor } from "../utils";
const { admin } = useComposition();
defineProps({
  logout: Function,
});

onMounted(() => {
  const list = MDCList.attachTo(document.querySelector(".mdc-deprecated-list"));
  list.wrapFocus = true;
  const header = document.querySelector(".mdc-drawer__header-img");
  var img = new Image();
  img.crossOrigin = "Anonymous";
  img.src =
    "https://cdn.discordapp.com/avatars/" +
    admin.value.id +
    "/" +
    admin.value.avatar +
    ".png";
  img.onload = function () {
    var rgb = averageColor(img);
    header.style.backgroundColor =
      "rgb(" + rgb.r + "," + rgb.g + "," + rgb.b + ")";
  };
});
</script>

<template>
  <aside class="mdc-drawer">
    <div
      class="mdc-drawer__header-img"
      :style="{
        backgroundImage:
          'url(https://cdn.discordapp.com/avatars/' +
          admin.id +
          '/' +
          admin.avatar +
          '.png)',
      }"
    ></div>
    <div class="mdc-drawer__header">
      <h3 class="mdc-drawer__title">
        {{ admin.name }}
      </h3>
      <h6 class="mdc-drawer__subtitle">{{ getRankName(admin.rank) }}</h6>
    </div>
    <div class="mdc-drawer__content">
      <nav class="mdc-deprecated-list">
        <li role="separator" class="mdc-deprecated-list-divider"></li>
        <router-link
          to="/"
          class="mdc-deprecated-list-item"
          aria-current="page"
        >
          <span class="mdc-deprecated-list-item__ripple"></span>
          <i
            class="material-icons mdc-deprecated-list-item__graphic"
            aria-hidden="true"
            >dashboard</i
          >
          <span class="mdc-deprecated-list-item__text">Dashboard</span>
        </router-link>
        <router-link
          to="/ranks"
          class="mdc-deprecated-list-item"
          aria-current="page"
        >
          <span class="mdc-deprecated-list-item__ripple"></span>
          <i
            class="material-icons mdc-deprecated-list-item__graphic"
            aria-hidden="true"
            >military_tech</i
          >
          <span class="mdc-deprecated-list-item__text">Ranks</span>
        </router-link>
        <router-link to="/events" class="mdc-deprecated-list-item">
          <span class="mdc-deprecated-list-item__ripple"></span>
          <i
            class="material-icons mdc-deprecated-list-item__graphic"
            aria-hidden="true"
            >calendar_today</i
          >
          <span class="mdc-deprecated-list-item__text">Events</span>
        </router-link>
        <li role="separator" class="mdc-deprecated-list-divider"></li>
        <a v-on:click="logout" class="mdc-deprecated-list-item">
          <span class="mdc-deprecated-list-item__ripple"></span>
          <i
            class="material-icons mdc-deprecated-list-item__graphic"
            aria-hidden="true"
            >logout</i
          >
          <span class="mdc-deprecated-list-item__text">Logout</span>
        </a>
      </nav>
    </div>
  </aside>
</template>

<style lang="scss" scoped>
@use "@material/button";
@use "@material/drawer";
@use "@material/list/mdc-list";

@import "../assets/shadows.scss";

// @include drawer.core-styles;
// @include list.deprecated-core-styles;

aside {
  @include box_shadow(2, false);
}

.mdc-drawer__header-img {
  display: flex;
  flex-direction: column;
  justify-content: end;
  min-height: 164px;
  border-radius: 0;

  background-position: center;
  background-repeat: no-repeat;
  background-size: 50%;
}

.router-link-exact-active {
  @extend .mdc-deprecated-list-item--activated;
}
</style>

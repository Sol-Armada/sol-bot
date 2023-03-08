<script setup>
import { useComposition } from "../compositions";
const { admin } = useComposition();
defineProps({
  logout: Function,
});
</script>

<template>
  <nav>
    <ul class="admin">
      <li>
        <img
          class="avatar"
          :src="`https://cdn.discordapp.com/avatars/${admin.id}/${admin.discord.user.avatar}.png`"
          alt=""
        />
      </li>
    </ul>

    <ul>
      <li>
        <router-link to="/ranks">
          <span class="material-symbols-outlined nav-icon">military_tech</span>
          <span class="nav-text"> Ranks </span>
        </router-link>
      </li>
      <li>
        <router-link to="/events">
          <span class="material-symbols-outlined nav-icon">calendar_month</span>
          <span class="nav-text"> Events </span>
        </router-link>
      </li>
    </ul>

    <ul class="logout">
      <li>
        <a v-on:click="logout">
          <span class="material-symbols-outlined nav-icon">logout</span>
          <span class="nav-text"> Logout </span>
        </a>
      </li>
    </ul>
  </nav>
</template>

<style lang="scss">
@import "../assets/variables.scss";
@import "../assets/shadows.scss";

nav {
  background: #212121;
  position: sticky;
  top: 0;
  bottom: 0;
  height: 100vh;
  left: 0;
  width: $nav-width;
  overflow: hidden;
  -webkit-transition: width 0.05s linear;
  transition: width 0.05s linear;
  -webkit-transform: translateZ(0) scale(1, 1);
  z-index: 1000;

  @include box_shadow(5, false);

  > ul {
    margin: 7px 0;

    &.admin {
      text-align: center;
      margin: 30px 0;

      img {
        width: 50%;
        border-radius: 100%;
      }
    }

    &.logout {
      position: absolute;
      left: 0;
      bottom: 0;
    }
  }

  li {
    position: relative;
    display: block;
    width: $nav-width;

    > a {
      position: relative;
      display: table;
      border-collapse: collapse;
      border-spacing: 0;
      color: #999;
      font-family: arial;
      font-size: 14px;
      text-decoration: none;
      -webkit-transform: translateZ(0) scale(1, 1);
      -webkit-transition: all 0.1s linear;

      > i {
        width: $nav-width;
        height: 36px;
        position: relative;
        text-align: center;
        vertical-align: middle;
        display: table-cell;
        font-size: 30px;
      }
    }
  }

  .nav-icon {
    position: relative;
    display: table-cell;
    width: 60px;
    height: 36px;
    text-align: center;
    vertical-align: middle;
    font-size: 18px;
  }

  .nav-text {
    position: relative;
    display: table-cell;
    vertical-align: middle;
    width: 190px;
    font-family: "Titillium Web", sans-serif;
  }
}

a:hover,
a:focus {
  text-decoration: none;
}

nav {
  -webkit-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  -o-user-select: none;
  user-select: none;
}

nav ul,
nav li {
  outline: 0;
  margin: 0;
  padding: 0;
}

.main-menu li:hover > a,
nav.main-menu li.active > a,
.dropdown-menu > li > a:hover,
.dropdown-menu > li > a:focus,
.dropdown-menu > .active > a,
.dropdown-menu > .active > a:hover,
.dropdown-menu > .active > a:focus,
.no-touch .dashboard-page nav.dashboard-menu ul li:hover a,
.dashboard-page nav.dashboard-menu ul li.active a {
  color: #fff;
  background-color: #5fa2db;
}
</style>

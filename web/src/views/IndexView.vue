<script setup>
    import { ref } from 'vue'
    const discordAuthUrl = ref(import.meta.env.VITE_DISCORD_AUTH_URL)

</script>

<template>
    <div class="login">
        <h1>Sol Armada Administration Portal</h1>
        <a :href="`${discordAuthUrl}`" class="mdc-button mdc-button--raised mdc-button--leading">
            <span class="mdc-button__ripple"></span>
            <i class="material-icons mdc-button__icon" aria-hidden="true">discord</i>
            <span class="mdc-button__label">Login with Discord</span>
        </a>
    </div>
</template>

<style lang="scss">
@use "@material/button/styles";
@use "@material/button/mixins";

.login {
    grid-row-start: 2;
    justify-items: center;
    align-items: center;
    text-align: center;

    a {
        &.mdc-button {
            @extend .mdc-button;
        }

        &.mdc-button--raised {
            @extend .mdc-button--raised;
            @include mixins.filled-accessible(#5865F2) // background-color: #5865F2!important;
        }

        &.mdc-button--leading {
            @extend .mdc-button--leading;
        }
    }
}
</style>

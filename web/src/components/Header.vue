<script setup>
import { useComposition } from '@/compositions'
import { useRouter } from 'vue-router'
import cookie from "@point-hub/vue-cookie"
const { user } = useComposition()
const router = useRouter()


function logout() {
    user.value = undefined
    cookie.remove("user")
    router.push("/")
}
</script>

<template>
    <header>
        <img class="avatar" :src="`https://cdn.discordapp.com/avatars/${user.id}/${user.avatar}.png`" alt="">
        <button class="mdc-button mdc-button--raised mdc-button--leading" @click="logout()">
            <span class="mdc-button__ripple"></span>
            <i class="material-icons mdc-button__icon" aria-hidden="true">logout</i>
            <span class="mdc-button__label">Logout</span>
        </button>
    </header>
</template>

<style lang="scss">
@use "@material/button/styles";

header {
    display: flex;
    align-items: center;
}

img.avatar {
    width: 50px;
    border-radius: 100%;
    margin: 10px;
}

button {
    justify-self: end;
}
</style>

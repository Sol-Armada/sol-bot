<script setup>
import Nav from "./components/Nav.vue"
import { onMounted } from "vue"
import { RouterView, useRouter } from "vue-router"
import { useComposition } from "./compositions"
import cookie from "@point-hub/vue-cookie"
const { admin } = useComposition()
const router = useRouter()

function logout() {
    admin.value = undefined
    cookie.remove("admin")
    router.push("/")
}
</script>

<template>
    <Nav v-if="admin" :admin="admin" :logout="logout" />
    <div class="content">
        <RouterView />
    </div>
</template>

<style scoped>

</style>

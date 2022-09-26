<template>
    <Header />
    <div class="admin">
        <List :users="test"/>
    </div>
</template>

<script>
import { useRouter } from 'vue-router'
import { useComposition } from '@/compositions'
import cookie from "@point-hub/vue-cookie"
import List from "../components/List.vue"
import Header from "../components/Header.vue"
import {ref} from 'vue'

export default {
    name: "Index",
    setup() {
        const { user } = useComposition()
        const router = useRouter()

        var cookieUser = undefined
        if (cookie.get("user") != undefined) {
            cookieUser = JSON.parse(cookie.get("user"))
        }

        if ((user.value == undefined || user.value.username == '') && !cookieUser) {
            router.push("/")
        }

        if (user.value == undefined && cookieUser) {
            user.value = cookieUser
        }

        const test = ref(["a","b","c","d"])

        return { user, test }
    },
    components: {
        List,
        Header
    }
}
</script>

<style>
.admin {
    grid-row-start: 2;
}
</style>

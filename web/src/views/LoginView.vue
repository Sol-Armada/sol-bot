<script setup>
import { useRoute, useRouter } from "vue-router"
import { useComposition } from "@/compositions"
import cookie from "@point-hub/vue-cookie"
import axios from "axios"
import { onMounted, ref } from "vue"
const route = useRoute()
const userCode = ref(route.query.code)
const { user } = useComposition()

onMounted(() => {
    const router = useRouter()
    const { err, user } = useComposition()

    var cookieUser = undefined
    if (cookie.get("user") != undefined) {
        cookieUser = JSON.parse(cookie.get("user"))
    }
    if (user.value == undefined && cookieUser) {
        // router.push("/admin")
    }

    if (userCode.value != undefined) {
        axios.post(`${import.meta.env.VITE_API_BASE_URL}/login`, {
            code: userCode.value
        }, {
            headers: {
                "Content-Type": "application/json"
            }
        })
            .then(resp => {
                user.value = resp.data
                setTimeout(() => {
                    cookie.set("user", JSON.stringify(resp.data))
                    router.push("/admin")
                }, 2000)
            })
            .catch(error => {
                if (error != undefined) {
                    if (error.message.includes("401")) {
                        err.value = 401
                        router.push("/error")
                    } else {
                        console.log(error)
                    }
                }
            })
    } else {
        router.push("/")
        console.log("don't ahve usercode")
    }
})
</script>

<template>
    <div class="logging-in">
        <h1 v-if="user">Welcome to Sol Armada Administration, {{user.username}}#{{user.discriminator}}</h1>
        <div class="lds-dual-ring" v-else></div>
    </div>
</template>

<style>
.logging-in {
    grid-row-start: 2;
}

.lds-dual-ring {
    display: inline-block;
    width: 80px;
    height: 80px;
}

.lds-dual-ring:after {
    content: " ";
    display: block;
    width: 64px;
    height: 64px;
    margin: 8px;
    border-radius: 50%;
    border: 6px solid #fff;
    border-color: #fff transparent #fff transparent;
    animation: lds-dual-ring 1.2s linear infinite;
}

@keyframes lds-dual-ring {
    0% {
        transform: rotate(0deg);
    }

    100% {
        transform: rotate(360deg);
    }
}
</style>

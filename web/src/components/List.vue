<script setup>
import { onMounted } from 'vue';

defineProps({
    users: Array,
})

onMounted(() => {
    let users = document.querySelectorAll(".columns .column .user")
    let columns = document.querySelectorAll(".columns .column")
    let ghost = document.createElement("div")
    ghost.classList.add("user")
    ghost.classList.add("ghost")

    function handleDragStart(e) {
        this.style.opacity = "0.4"

        e.dataTransfer.effectAllowed = "move"
        e.dataTransfer.setData("text/plain", e.target.dataset.id)
    }
    function handleDragEnd(e) {
        this.style.opacity = "1"

        users.forEach(function (user) {
            user.classList.remove("over")
        })
    }
    function handleDragOver(e) {
        e.preventDefault()
        return false
    }
    function handleDragEnter(e) {
        this.appendChild(ghost)
    }
    function handleDragLeave(e) {
        this.classList.remove("over")
    }
    function handleDrop(e) {
        e.stopPropagation()
        
        ghost.remove()

        var usersList = document.querySelectorAll(".user")
        for (var i = 0; i < usersList.length; i++) {
            var user = usersList[i]
            if (user.dataset.id === e.dataTransfer.getData("text/plain")) {
                this.appendChild(user)
                break
            }
        }

        return false
    }

    // add listeners
    users.forEach(function (user) {
        user.addEventListener("dragstart", handleDragStart)
        user.addEventListener("dragend", handleDragEnd)
    })
    columns.forEach(function (column) {
        column.addEventListener("dragover", handleDragOver)
        column.addEventListener("dragenter", handleDragEnter)
        column.addEventListener("dragleave", handleDragLeave)
        column.addEventListener("drop", handleDrop)
    })
})
</script>

<template>
    <div class="columns">
        <div class="column">
            <div class="title">Recruit</div>
            <div class="user" draggable="true" v-for="user in users" :data-id="`${user}`">{{user}}</div>
        </div>
        <div class="column">
            <div class="title">Member</div>
        </div>
        <div class="column">
            <div class="title">Technician</div>
        </div>
        <div class="column">
            <div class="title">Specialist</div>
        </div>
    </div>
</template>

<style>
.columns {
    grid-column-start: 2;
    display: grid;
    grid-template-columns: 1fr 1fr 1fr 1fr;
    gap: 10px;
    height: 100%;
}

.columns .column {
    height: 100%;
}

.title {
    padding: 10px;
}

.user {
    margin: 10px;
    color: white;
    background-color: grey;
    border-radius: .5em;
    padding: 10px;
    cursor: move;
}

.ghost {
    border: 3px dotted white;
    opacity: "0.8";
}
</style>

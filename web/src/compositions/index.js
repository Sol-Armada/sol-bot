import { ref } from 'vue'

const user = ref()
const err = ref()

export const useComposition = function() {
    return { user, err }
}

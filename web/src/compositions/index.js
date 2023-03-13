import { ref } from "vue";

const admin = ref();
const users = ref();
const events = ref();
const err = ref();
const bank = ref();

export const useComposition = function () {
  return { admin, users, events, bank, err };
};

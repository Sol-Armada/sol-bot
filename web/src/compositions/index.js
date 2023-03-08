import { ref } from "vue";

const admin = ref();
const users = ref();
const events = ref();
const err = ref();

export const useComposition = function () {
  return { admin, users, events, err };
};

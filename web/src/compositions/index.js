import { ref, watch } from "vue";

const admin = ref();
const users = ref();
const events = ref();
const err = ref();
const bank = ref();

watch(events, (newEvents) => {
  newEvents.sort((a, b) => a.start - b.start);
});

export const useComposition = function () {
  return { admin, users, events, bank, err };
};

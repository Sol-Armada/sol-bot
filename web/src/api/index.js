import axios from "axios";
import { useComposition } from "../compositions";

export function getUser(userId) {
  const { admin } = useComposition();
  axios
    .get(`${import.meta.env.VITE_API_BASE_URL}/users/${userId}`, {
      headers: {
        "Content-Type": "application/json",
        "X-User-Id": admin.value.id,
      },
    })
    .then((resp) => {
      return resp.data;
    })
    .catch((err) => {
      console.error(err);
    });
}

export function updateUser(user) {
  const { admin } = useComposition();
  axios
    .put(
      `${import.meta.env.VITE_API_BASE_URL}/users/${user.id}`,
      {
        user: user,
      },
      {
        headers: {
          "Content-Type": "application/json",
          "X-User-Id": admin.value.id,
        },
      }
    )
    .then(() => {
      getUsers();
    })
    .catch((err) => {
      console.error(err);
    });
}

export function getUsers() {
  const { admin, users } = useComposition();
  axios
    .get(`${import.meta.env.VITE_API_BASE_URL}/users/`, {
      headers: {
        "X-User-Id": admin.value.id,
      },
    })
    .then((resp) => {
      var u = resp.data;
      u.sort((a, b) => {
        if (a.rank > b.rank) {
          return 1;
        } else if (a.rank < b.rank) {
          return -1;
        }

        const aName = a.name.toUpperCase();
        const bName = b.name.toUpperCase();

        if (aName < bName) {
          return -1;
        }

        if (aName > bName) {
          return 1;
        }

        return 0;
      });
      users.value = u;
    })
    .catch((err) => {
      console.error(err);
      users.value = [];
    });
}

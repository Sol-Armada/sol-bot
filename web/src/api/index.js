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
    .catch((err) => {
      console.error(err);
    });
}

export const getUsers = new Promise((resolve, reject) => {
  axios
    .get(`${import.meta.env.VITE_API_BASE_URL}/users/`)
    .then((resp) => {
      var u = resp.data.users;
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
      // users.value = u;
      resolve(u);
    })
    .catch((err) => {
      reject(err);
    });
});

export const getBankBalance = new Promise((resolve, reject) => {
  axios
    .get(`${import.meta.env.VITE_API_BASE_URL}/bank/balance`, {
      headers: {
        "Content-Type": "application/json",
      },
    })
    .then((resp) => {
      resolve(resp.data.balance);
    })
    .catch(reject);
});

export const getEvents = new Promise((resolve, reject) => {
  axios
    .get(`${import.meta.env.VITE_API_BASE_URL}/events/`)
    .then((resp) => {
      var e = resp.data.events;

      e.sort((a, b) => {
        if (new Date(a.start).getTime() < new Date(b.start).getTime()) {
          return -1;
        } else if (new Date(a.start).getTime() > new Date(b.start).getTime()) {
          return 1;
        } else {
          return 0;
        }
      });

      resolve(e);
    })
    .catch(reject);
});

export function createEvent(event) {
  return new Promise((resolve, reject) => {
    var originalPos = event.positions;
    var obj = {};
    event.positions.forEach(function (value, key) {
      obj[key] = value;
    });
    event.positions = obj;
    axios
      .post(
        `${import.meta.env.VITE_API_BASE_URL}/events`,
        JSON.stringify(event),
        {
          headers: {
            "Content-Type": "application/json",
          },
        }
      )
      .then((resp) => {
        resolve(resp.data.event);
      })
      .catch(reject);
    event.positions = originalPos;
  });
}

export const updateEvent = function (event) {
  return new Promise((resolve, reject) => {
    axios;
    event.positions = event.positionsMap;
    axios
      .put(`${import.meta.env.VITE_API_BASE_URL}/events/${event._id}`, event, {
        headers: {
          "Content-Type": "application/json",
        },
      })
      .then((resp) => {
        var updatedEvent = resp.data.event;
        resolve(updatedEvent);
      })
      .catch(reject);
  });
};

export function deleteEvent(id) {
  const { events } = useComposition();
  axios
    .delete(`${import.meta.env.VITE_API_BASE_URL}/events/${id}`, {
      headers: {
        "Content-Type": "application/json",
      },
    })
    .then(() => {
      var e = events.value;
      for (let index = 0; index < e.length; index++) {
        const ev = e[index];
        if (ev._id == id) {
          e.splice(index, 1);
          events.value = e;
          break;
        }
      }
    })
    .catch((err) => {
      console.error(err);
    });
}

export const getRandomNames = function (max, rankLimit) {
  return new Promise((resolve, reject) => {
    let names = "";
    if (max > 0 && rankLimit != undefined) {
      axios
        .get(
          `${
            import.meta.env.VITE_API_BASE_URL
          }/users/random?max=${max}&rank_limit=${rankLimit}`
        )
        .then((resp) => {
          resp.data.users.forEach((user, i) => {
            names += "- " + user.name;
            if (resp.data.users.length != i + 1) {
              names += "\n";
            }
          });
          resolve(names);
        })
        .catch(reject);
    }
  });
};

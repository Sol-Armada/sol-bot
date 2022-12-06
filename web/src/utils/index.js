export function truncateString(string, limit) {
  if (string.length > limit) {
    return string.substring(0, limit) + "...";
  } else {
    return string;
  }
}

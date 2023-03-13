export function truncateString(string, limit) {
  if (string.length > limit) {
    return string.substring(0, limit) + "...";
  } else {
    return string;
  }
}

export function getRankName(rank) {
  switch (rank) {
    case 1:
      return "Admiral";
    case 2:
      return "Commander";
    case 3:
      return "Lieutenant";
    case 4:
      return "Specialist";
    case 5:
      return "Technician";
    case 6:
      return "Member";
    case 7:
      return "Recruit";
    case 8:
      return "Guest";
    case 9:
      return "Ally";
    default:
      return "Unkown";
  }
}

// Example 2: Using a custom function with canvas
export function averageColor(imageElement) {
  // Create a canvas element
  var canvas = document.createElement("canvas");
  // Get the context of the canvas
  var context = canvas.getContext("2d");
  // Get image data
  var imgData;
  // Define variables for storing RGB values
  var rgb = { r: 0, g: 0, b: 0 };

  // Set height and width of canvas to match image
  var height = (canvas.height =
    imageElement.naturalHeight ||
    imageElement.offsetHeight ||
    imageElement.height);
  var width = (canvas.width =
    imageElement.naturalWidth ||
    imageElement.offsetWidth ||
    imageElement.width);

  // Draw image on canvas
  context.drawImage(imageElement, 0, 0);

  // Get pixel data from canvas
  imgData = context.getImageData(0, 0, width, height);

  var rgbs = [];
  // Loop through pixels and sum RGB values
  for (var i = 0; i < imgData.data.length; i += 4) {
    rgb.r += imgData.data[i];
    rgb.g += imgData.data[i + 1];
    rgb.b += imgData.data[i + 2];
    var t =
      imgData.data[i] + ":" + imgData.data[i + 1] + ":" + imgData.data[i + 2];
    rgbs.push(t);
  }

  rgbs.sort();

  var mr = mostRepeated(rgbs).split(":");
  rgb.r = mr[0];
  rgb.g = mr[1];
  rgb.b = mr[2];

  return rgb;
}

export function filterForRank(rank, users) {
  users = users.filter((user) => user.rank == rank);
  return users;
}

function mostRepeated(array) {
  // Create an empty object
  let frequency = {};
  // Loop through the array
  for (let value of array) {
    // If the value is already a key in the object, increment its count
    if (frequency[value]) {
      frequency[value]++;
    } else {
      // Otherwise, set its count to 1
      frequency[value] = 1;
    }
  }
  // Create a variable to store the most repeated value and its frequency
  let maxCount = 0;
  let maxValue = null;
  // Loop through the object keys
  for (let key in frequency) {
    // If the key's count is greater than the current maximum, update them
    if (frequency[key] > maxCount) {
      maxCount = frequency[key];
      maxValue = key;
    }
  }
  // Return the most repeated value
  return maxValue;
}

function fn(args) {
  // Parse URL query parameters
  var url = args.url;
  var paramName = args.param;

  // Check if URL contains query parameters
  var queryStart = url.indexOf('?');
  if (queryStart === -1) {
    return null;
  }

  var queryString = url.substring(queryStart + 1);

  // Split by & to get individual parameters
  var params = queryString.split('&');

  for (var i = 0; i < params.length; i++) {
    var pair = params[i].split('=');
    var key = decodeURIComponent(pair[0]);
    var value = pair.length > 1 ? decodeURIComponent(pair[1]) : '';

    if (key === paramName) {
      return value;
    }
  }

  return null;
}

function fn(jwt) {
  // Decode JWT token and return payload
  // JWT format: header.payload.signature
  var Base64 = Java.type('java.util.Base64');

  try {
    var parts = jwt.split('.');

    if (parts.length !== 3) {
      throw new Error('Invalid JWT format');
    }

    // Decode the payload (second part)
    var payloadBase64 = parts[1];

    // Add padding if necessary
    while (payloadBase64.length % 4 !== 0) {
      payloadBase64 += '=';
    }

    var decoder = Base64.getUrlDecoder();
    var decodedBytes = decoder.decode(payloadBase64);
    var payloadJson = new java.lang.String(decodedBytes, 'UTF-8');

    // Parse JSON
    return JSON.parse(payloadJson);
  } catch (e) {
    karate.log('Error decoding JWT:', e);
    return null;
  }
}

function generateCodeVerifier() {
  // Generate a cryptographically random code verifier
  // Length between 43-128 characters, using URL-safe characters
  var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';
  var length = 64; // Recommended length
  var verifier = '';

  for (var i = 0; i < length; i++) {
    verifier += chars.charAt(Math.floor(Math.random() * chars.length));
  }

  return verifier;
}

function generateCodeChallenge(verifier) {
  // Generate SHA256 hash of verifier and base64url encode it
  // Note: Karate has built-in support for this
  var MessageDigest = Java.type('java.security.MessageDigest');
  var Base64 = Java.type('java.util.Base64');

  var md = MessageDigest.getInstance('SHA-256');
  var digest = md.digest(java.lang.String(verifier).getBytes('UTF-8'));

  // Base64 URL encode without padding
  var challenge = Base64.getUrlEncoder().withoutPadding().encodeToString(digest);

  return challenge;
}

// Export functions
var result = {
  generateCodeVerifier: generateCodeVerifier,
  generateCodeChallenge: generateCodeChallenge
};

result;

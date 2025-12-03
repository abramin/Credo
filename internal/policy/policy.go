package policy

import "time"

// RegistryCacheTTL enforces retention for sensitive registry data. In regulated
// environments this should be short; adjust before production use.
var RegistryCacheTTL = 5 * time.Minute

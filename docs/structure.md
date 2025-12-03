internal/
  platform/
    config/
    logger/
    httpserver/
  auth/
    service.go       // Users, sessions
    store.go         // UserStore, SessionStore
    models.go
  consent/
    service.go       // Grant, revoke, RequireConsent
    store.go
    models.go
  evidence/
    registry/
      client_citizen.go
      client_sanctions.go
      service.go     // RegistryService, caching, minimisation
      store.go       // RegistryCacheStore
    vc/
      service.go     // VCService
      store.go
      models.go
  decision/
    service.go       // Evaluate
    store.go
    models.go
  audit/
    publisher.go     // channel or queue producer
    worker.go        // background consumer
    store.go
    models.go
  transport/
    http/
      handlers_auth.go
      handlers_consent.go
      handlers_evidence.go
      handlers_decision.go
      handlers_me.go
      router.go

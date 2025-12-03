package test

// import (
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"

// func TestRouterScaffold(t *testing.T) {
// 	bdd.Given(t, "the HTTP router", func(t *testing.T) {
// 		handler := httptransport.NewHandler(false)
// 		router := httptransport.NewRouter(handler)

// 		bdd.When(t, "calling POST /auth/authorize", func(t *testing.T) {
// 			req := httptest.NewRequest(http.MethodPost, "/auth/authorize", nil)
// 			rec := httptest.NewRecorder()

// 			router.ServeHTTP(rec, req)

// 			bdd.Then(t, "it should respond with not implemented", func(t *testing.T) {
// 				if rec.Code != http.StatusNotImplemented {
// 					t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, rec.Code)
// 				}
// 			})
// 		})

// 		bdd.When(t, "calling GET /me/data-export", func(t *testing.T) {
// 			req := httptest.NewRequest(http.MethodGet, "/me/data-export", nil)
// 			rec := httptest.NewRecorder()

// 			router.ServeHTTP(rec, req)

// 			bdd.Then(t, "it should respond with not implemented", func(t *testing.T) {
// 				if rec.Code != http.StatusNotImplemented {
// 					t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, rec.Code)
// 				}
// 			})
// 		})
// 	})
// }

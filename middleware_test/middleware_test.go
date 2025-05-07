// middleware/auth_test.go
package middleware_test

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func init() {
	// Set test JWT secret
	config.JWTSecret = []byte("test-secret")
}

func TestAuthMiddleware(t *testing.T) {
	// Create a test handler that just returns 200 OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user claims are in the context
		claims, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			t.Error("Expected user claims in context")
		}

		// Verify the claims
		if claims.ID != "test-id" || claims.Username != "testuser" {
			t.Errorf("Claims mismatch: got ID=%s, Username=%s, want ID=test-id, Username=testuser",
				claims.ID, claims.Username)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create a JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       "test-id",
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(config.JWTSecret)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// Create a request with the token
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Record the response
	rr := httptest.NewRecorder()

	// Wrap the test handler with the auth middleware
	middleware.AuthMiddleware(testHandler).ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestAuthMiddlewareMissingToken(t *testing.T) {
	// Create a test handler that should not be called
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when token is missing")
	})

	// Create a request without a token
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Wrap the test handler with the auth middleware
	middleware.AuthMiddleware(testHandler).ServeHTTP(rr, req)

	// Check the status code (should be 401 Unauthorized)
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	// Create a test handler that should not be called
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when token is invalid")
	})

	// Create a request with an invalid token
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer invalid-token")

	// Record the response
	rr := httptest.NewRecorder()

	// Wrap the test handler with the auth middleware
	middleware.AuthMiddleware(testHandler).ServeHTTP(rr, req)

	// Check the status code (should be 401 Unauthorized)
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}

func TestGetUserFromContext(t *testing.T) {
	// Create user claims
	expectedClaims := middleware.UserClaims{
		ID:       "test-id",
		Username: "testuser",
	}

	// Create a context with the claims
	ctx := context.WithValue(context.Background(), middleware.UserContextKey, expectedClaims)

	// Get the claims from the context
	claims, ok := middleware.GetUserFromContext(ctx)

	// Verify the result
	if !ok {
		t.Error("Expected to get claims from context")
	}

	if claims.ID != expectedClaims.ID || claims.Username != expectedClaims.Username {
		t.Errorf("Claims mismatch: got %+v, want %+v", claims, expectedClaims)
	}
}

func TestGetUserFromContextMissing(t *testing.T) {
	// Create a context without claims
	ctx := context.Background()

	// Try to get claims from the context
	_, ok := middleware.GetUserFromContext(ctx)

	// Verify the result
	if ok {
		t.Error("Expected to not get claims from context")
	}
}

func TestCORSMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Wrap the test handler with the CORS middleware
	middleware.CORSMiddleware(testHandler).ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the CORS headers
	expectedOrigin := "http://localhost:4200"
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != expectedOrigin {
		t.Errorf("handler returned wrong CORS origin header: got %v want %v", origin, expectedOrigin)
	}

	expectedMethods := "GET, POST, PUT, DELETE, OPTIONS"
	if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods != expectedMethods {
		t.Errorf("handler returned wrong CORS methods header: got %v want %v", methods, expectedMethods)
	}
}

func TestCORSMiddlewareOptions(t *testing.T) {
	// Create a test handler that should not be called for OPTIONS requests
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			t.Error("Handler should not be called for OPTIONS request")
		}
	})

	// Create an OPTIONS request
	req, err := http.NewRequest("OPTIONS", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Wrap the test handler with the CORS middleware
	middleware.CORSMiddleware(testHandler).ServeHTTP(rr, req)

	// Check the status code (should be 200 OK for preflight)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the CORS headers
	expectedOrigin := "http://localhost:4200"
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != expectedOrigin {
		t.Errorf("handler returned wrong CORS origin header: got %v want %v", origin, expectedOrigin)
	}
}

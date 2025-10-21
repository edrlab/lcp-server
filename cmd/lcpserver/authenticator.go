// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/golang-jwt/jwt/v5"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// validateCredentials checks if the provided username and password match
// any of the configured admin accounts
func validateCredentials(username, password string, config *conf.Config) bool {
	if storedPassword, exists := config.JWT.Admin[username]; exists {
		return storedPassword == password
	}
	return false
}

// Login creates a login handler using the provided configuration
func Login(config *conf.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check credentials using configured admin accounts
		if !validateCredentials(creds.Username, creds.Password, config) {
			log.Printf("🚫 Connection attempt failed for user: %s", creds.Username)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Println("🔐 User logged in:", creds.Username)

		// Create JWT token using the configured secret key
		expirationTime := time.Now().Add(1 * time.Hour)
		claims := &Claims{
			Username: creds.Username,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(config.JWT.SecretKey))
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Send back the token in a cookie (optional)
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    tokenString,
			Expires:  expirationTime,
			HttpOnly: true,
			Secure:   false, // false for local development
			SameSite: http.SameSiteStrictMode,
		})

		// Also return the token and user information as JSON
		response := map[string]interface{}{
			"token": tokenString,
			"user": map[string]interface{}{
				"id":    "1",
				"email": creds.Username + "@example.com",
				"name":  creds.Username,
			},
		}

		responseJSON, _ := json.Marshal(response)
		log.Println("Response:", string(responseJSON))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AuthMiddleware creates JWT authentication middleware using the provided configuration
func AuthMiddleware(config *conf.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string

			// Try to get a token from the Authorization header first (Bearer token)
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenStr = authHeader[7:]
			} else {
				// Fallback to cookie if no Authorization header
				c, err := r.Cookie("token")
				if err != nil {
					if err == http.ErrNoCookie {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						json.NewEncoder(w).Encode(map[string]string{"error": "No authentication token provided"})
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "Bad request"})
					return
				}
				tokenStr = c.Value
			}

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(config.JWT.SecretKey), nil
			})

			if err != nil {
				log.Println("JWT parse error:", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)

				// Handle JWT validation errors using modern Go error handling
				errorMessage := "Invalid token"
				var errorCode string

				// Use errors.Is() to check for specific JWT errors in composite errors
				if errors.Is(err, jwt.ErrTokenExpired) {
					errorMessage = "Token has expired"
					errorCode = "TOKEN_EXPIRED"
				} else if errors.Is(err, jwt.ErrSignatureInvalid) { // msg is "token signature is invalid"
					errorMessage = "Invalid token signature"
				} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
					errorMessage = "Token not valid yet"
				} else if errors.Is(err, jwt.ErrTokenMalformed) {
					errorMessage = "Token is malformed"
				} else if errors.Is(err, jwt.ErrTokenUnverifiable) {
					errorMessage = "Token could not be verified"
				} else {
					// For any other JWT parsing error, provide a generic message
					errorMessage = "Invalid or malformed token"
				}

				response := map[string]string{"error": errorMessage}
				if errorCode != "" {
					response["code"] = errorCode
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			if !token.Valid {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Token is not valid"})
				return
			}

			// Add username to request context for use in handlers
			r.Header.Set("X-Username", claims.Username)

			next.ServeHTTP(w, r)
		})
	}
}

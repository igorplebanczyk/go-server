package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"strings"
)

func (cfg *apiConfig) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		ID          int    `json:"id"`
		Email       string `json:"email"`
		IsChirpyRed bool   `json:"is_chirpy_red"`
	}

	userID, err := cfg.GetAuthenticatedUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Failed to authenticate user")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if !strings.Contains(params.Email, "@") || !strings.Contains(params.Email, ".") {
		respondWithError(w, http.StatusBadRequest, "Invalid email")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	err = cfg.db.UpdateUser(userID, params.Email, hashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	user, err := cfg.db.GetUserByID(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		ID:          user.ID,
		Email:       params.Email,
		IsChirpyRed: user.IsChirpyRed,
	})
}

func (cfg *apiConfig) GetAuthenticatedUserID(r *http.Request) (int, error) {
	token, err := RetrieveTokenFromHeader(r)
	if err != nil {
		return -1, fmt.Errorf("no token provided")
	}

	parsedToken, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.jwtSecret), nil
	})
	if err != nil || !parsedToken.Valid {
		return -1, fmt.Errorf("invalid token")
	}

	claims, ok := parsedToken.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return -1, fmt.Errorf("invalid claims type")
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return -1, fmt.Errorf("failed to parse user ID")
	}

	return userID, nil
}

func RetrieveTokenFromHeader(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return "", errors.New("no token provided")
	}

	if !strings.HasPrefix(token, "Bearer ") {
		return "", errors.New("invalid token format")
	}

	token = strings.TrimPrefix(token, "Bearer ") // Trim the "Bearer " prefix to get the actual token
	if token == "" {
		return "", errors.New("malformed token")
	}

	return token, nil
}

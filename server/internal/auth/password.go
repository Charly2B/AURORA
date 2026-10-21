// Package auth : mots de passe (Argon2id), sessions et limitation des tentatives de connexion.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Paramètres Argon2id (RFC 9106, profil « mémoire réduite ») : 64 Mio, 3 passes, 2 fils.
// Raisonnable sur un Ryzen V1500B : ~100 ms par vérification, sans saturer la RAM du NAS.
const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
	saltLen      = 16
)

// MinPasswordLen : longueur minimale d'un mot de passe.
const MinPasswordLen = 8

// HashPassword renvoie un hachage au format PHC : $argon2id$v=19$m=65536,t=3,p=2$sel$hash
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLen {
		return "", fmt.Errorf("le mot de passe doit contenir au moins %d caractères", MinPasswordLen)
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemory, argonTime,
		argonThreads, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

// CheckPassword compare en temps constant. Les paramètres sont lus dans le hachage stocké,
// ce qui permet de les durcir plus tard sans invalider les comptes existants.
func CheckPassword(hash, password string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("format de hachage inconnu")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errors.New("version argon2 incompatible")
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, errors.New("paramètres argon2 illisibles")
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := enc.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// dummyHash sert à égaliser le temps de réponse quand l'utilisateur n'existe pas.
var dummyHash, _ = HashPassword("aurora-dummy-password")

// BurnTime effectue une vérification factice (anti-énumération des comptes par le temps de réponse).
func BurnTime(password string) { CheckPassword(dummyHash, password) }

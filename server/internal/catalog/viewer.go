// Package catalog : requêtes de lecture du catalogue, toujours filtrées par les droits du profil
// (bibliothèques autorisées, contrôle parental). Le filtrage est fait côté serveur, jamais dans l'interface.
package catalog

import (
	"strconv"
	"strings"
)

// Viewer : profil qui consulte le catalogue.
type Viewer struct {
	UserID       int64
	AllLibraries bool
	MaxAge       *int
	HideUnrated  bool
}

// Where renvoie une condition SQL (et ses paramètres) restreignant l'alias d'items aux contenus visibles.
// Le contrôle parental repose sur la classification TMDB : un contenu sans classification n'est masqué
// que si le profil le demande (hide_unrated). Ce n'est donc pas infaillible si les métadonnées manquent.
func (v Viewer) Where(alias string) (string, []any) {
	var conds []string
	var args []any
	if !v.AllLibraries {
		conds = append(conds, alias+".library_id IN (SELECT library_id FROM user_libraries WHERE user_id = ?)")
		args = append(args, v.UserID)
	}
	if v.MaxAge != nil {
		// Un épisode sans classification propre hérite de celle de sa série (sinon un épisode ajouté après
		// l'identification de la série échapperait au filtre).
		age := "COALESCE(" + alias + ".min_age, (SELECT par.min_age FROM items par WHERE par.id = " + alias + ".parent_id))"
		c := "(" + age + " IS NOT NULL AND " + age + " <= " + strconv.Itoa(*v.MaxAge) + ")"
		if !v.HideUnrated {
			c = "(" + age + " IS NULL OR " + c + ")"
		}
		conds = append(conds, c)
	}
	if len(conds) == 0 {
		return "1=1", nil
	}
	return strings.Join(conds, " AND "), args
}

// presentMovie : le film a au moins un fichier présent.
const presentMovie = `EXISTS (SELECT 1 FROM media_files f WHERE f.item_id = i.id AND f.present = 1)`

// presentSeries : la série a au moins un épisode avec un fichier présent.
const presentSeries = `EXISTS (SELECT 1 FROM items e JOIN media_files f ON f.item_id = e.id WHERE e.parent_id = i.id AND f.present = 1)`

package server

import (
	"fmt"
	"net/http"
	"os"
	"sort"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/pkg/swagger"
	swaggerui "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/swagger-ui"
)

func addSwagger(mux *http.ServeMux) {
	mux.HandleFunc("/swagger/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		_, span := global.Tracer.Start(r.Context(), "swagger")
		defer span.End()

		mergedSwagger := swagger.NewMerger()
		ds, err := os.ReadDir("./gen/api/v1")
		if err != nil {
			http.Error(w, "Reading generated swagger directories", http.StatusInternalServerError)
			return
		}
		sortDirs(ds)
		for _, d := range ds {
			swaggerPath := fmt.Sprintf("./gen/api/v1/%[1]s/%[1]s.swagger.json", d.Name())
			if err := mergedSwagger.AddFile(swaggerPath); err != nil {
				http.Error(w, "Merging swaggers", http.StatusInternalServerError)
				return
			}
		}

		b, err := mergedSwagger.MarshalJSON()
		if err != nil {
			http.Error(w, "Exporting merged swagger", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(b); err != nil {
			http.Error(w, "Writing merged swagger", http.StatusInternalServerError)
			return
		}
	})
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerui.Content))))
}

// sorts the directories in alphabetic order, and if provided, set
// the "api" directory last (should contain the swagger global infos).
func sortDirs(entries []os.DirEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		nameI := entries[i].Name()
		nameJ := entries[j].Name()

		// Check if either is the "api" directory
		isAPII := entries[i].IsDir() && nameI == "api"
		isAPIJ := entries[j].IsDir() && nameJ == "api"

		if isAPII && !isAPIJ {
			return false // i should come after j
		}
		if !isAPII && isAPIJ {
			return true // i should come before j
		}

		// Otherwise sort alphabetically
		return nameI < nameJ
	})
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// -----------------------------
// Domain types
// -----------------------------

type MovieInfo struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	Director    string   `json:"director"`
	Rating      float64  `json:"rating"`
	Source      string   `json:"source"`
	LastUpdated string   `json:"last_updated"`
}

// -----------------------------
// MovieDatabase & Interface
// -----------------------------

type MovieDB interface {
	Add(movie MovieInfo) error
	Get(id string) (*MovieInfo, error)
	Search(query string) ([]MovieInfo, error)
	GetByGenre(genre string) ([]MovieInfo, error)
	GetByYear(year int) ([]MovieInfo, error)
	GetByDirector(director string) ([]MovieInfo, error)
	Update(movie MovieInfo) error
	Delete(id string) error
	Save(filename string) error
	Load(filename string) error
}

type MovieDatabase struct {
	Movies      map[string]MovieInfo `json:"movies"`
	Genres      map[string][]string  `json:"genres"`
	Directors   map[string][]string  `json:"directors"`
	Years       map[int][]string     `json:"years"`
	LastUpdated time.Time            `json:"last_updated"`
	TotalCount  int                  `json:"total_count"`
}

func NewMovieDatabase() *MovieDatabase {
	return &MovieDatabase{
		Movies:    make(map[string]MovieInfo),
		Genres:    make(map[string][]string),
		Directors: make(map[string][]string),
		Years:     make(map[int][]string),
	}
}

// -----------------------------
// Helper index utilities
// -----------------------------

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}

// -----------------------------
// CRUD: Add / Get / Search / GetBy... / Update / Delete
// -----------------------------

func (db *MovieDatabase) Add(movie MovieInfo) error {
	if movie.ID == "" {
		return fmt.Errorf("movie must have an ID")
	}

	// If the movie already exists, treat Add as no-op (or you might prefer error)
	if _, exists := db.Movies[movie.ID]; exists {
		return fmt.Errorf("movie already exists: %s", movie.ID)
	}

	// Add to main map
	db.Movies[movie.ID] = movie

	// Update genre index
	for _, genre := range movie.Genres {
		if !containsString(db.Genres[genre], movie.ID) {
			db.Genres[genre] = append(db.Genres[genre], movie.ID)
		}
	}

	// Update director index
	if movie.Director != "" {
		if !containsString(db.Directors[movie.Director], movie.ID) {
			db.Directors[movie.Director] = append(db.Directors[movie.Director], movie.ID)
		}
	}

	// Update years index
	if movie.Year > 0 {
		if !containsString(db.Years[movie.Year], movie.ID) {
			db.Years[movie.Year] = append(db.Years[movie.Year], movie.ID)
		}
	}

	db.TotalCount++
	return nil
}

func (db *MovieDatabase) Get(id string) (*MovieInfo, error) {
	m, ok := db.Movies[id]
	if !ok {
		return nil, fmt.Errorf("movie not found: %s", id)
	}
	return &m, nil
}

func (db *MovieDatabase) Search(query string) ([]MovieInfo, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}
	results := []MovieInfo{}
	for _, m := range db.Movies {
		if strings.Contains(strings.ToLower(m.Title), q) || strings.Contains(strings.ToLower(m.Description), q) {
			results = append(results, m)
		}
	}
	return results, nil
}

func (db *MovieDatabase) GetByGenre(genre string) ([]MovieInfo, error) {
	results := []MovieInfo{}
	ids, ok := db.Genres[genre]
	if !ok {
		return results, nil
	}
	for _, id := range ids {
		if m, err := db.Get(id); err == nil {
			results = append(results, *m)
		}
	}
	return results, nil
}

func (db *MovieDatabase) GetByYear(year int) ([]MovieInfo, error) {
	results := []MovieInfo{}
	ids, ok := db.Years[year]
	if !ok {
		return results, nil
	}
	for _, id := range ids {
		if m, err := db.Get(id); err == nil {
			results = append(results, *m)
		}
	}
	return results, nil
}

func (db *MovieDatabase) GetByDirector(director string) ([]MovieInfo, error) {
	results := []MovieInfo{}
	ids, ok := db.Directors[director]
	if !ok {
		return results, nil
	}
	for _, id := range ids {
		if m, err := db.Get(id); err == nil {
			results = append(results, *m)
		}
	}
	return results, nil
}

func (db *MovieDatabase) Update(movie MovieInfo) error {
	if movie.ID == "" {
		return fmt.Errorf("movie must have an ID")
	}

	existing, ok := db.Movies[movie.ID]
	if !ok {
		return fmt.Errorf("movie does not exist: %s", movie.ID)
	}

	// If director changed, update director index
	if existing.Director != "" && existing.Director != movie.Director {
		db.Directors[existing.Director] = removeString(db.Directors[existing.Director], movie.ID)
	}
	if movie.Director != "" && !containsString(db.Directors[movie.Director], movie.ID) {
		db.Directors[movie.Director] = append(db.Directors[movie.Director], movie.ID)
	}

	// If year changed, update year index
	if existing.Year != 0 && existing.Year != movie.Year {
		db.Years[existing.Year] = removeString(db.Years[existing.Year], movie.ID)
	}
	if movie.Year != 0 && !containsString(db.Years[movie.Year], movie.ID) {
		db.Years[movie.Year] = append(db.Years[movie.Year], movie.ID)
	}

	// If genres changed, update genre index (remove old ones not present, add new ones)
	oldGenres := map[string]bool{}
	for _, g := range existing.Genres {
		oldGenres[g] = true
	}
	newGenres := map[string]bool{}
	for _, g := range movie.Genres {
		newGenres[g] = true
	}
	// remove old genres that are not in new
	for g := range oldGenres {
		if !newGenres[g] {
			db.Genres[g] = removeString(db.Genres[g], movie.ID)
		}
	}
	// add new genres
	for g := range newGenres {
		if !containsString(db.Genres[g], movie.ID) {
			db.Genres[g] = append(db.Genres[g], movie.ID)
		}
	}

	// Update main map
	db.Movies[movie.ID] = movie
	return nil
}

func (db *MovieDatabase) Delete(id string) error {
	movie, ok := db.Movies[id]
	if !ok {
		return fmt.Errorf("movie not found: %s", id)
	}

	// Remove from genre index
	for _, g := range movie.Genres {
		db.Genres[g] = removeString(db.Genres[g], id)
		if len(db.Genres[g]) == 0 {
			delete(db.Genres, g)
		}
	}

	// Remove from director index
	if movie.Director != "" {
		db.Directors[movie.Director] = removeString(db.Directors[movie.Director], id)
		if len(db.Directors[movie.Director]) == 0 {
			delete(db.Directors, movie.Director)
		}
	}

	// Remove from years index
	if movie.Year != 0 {
		db.Years[movie.Year] = removeString(db.Years[movie.Year], id)
		if len(db.Years[movie.Year]) == 0 {
			delete(db.Years, movie.Year)
		}
	}

	// Remove from main map
	delete(db.Movies, id)
	if db.TotalCount > 0 {
		db.TotalCount--
	}
	return nil
}

// -----------------------------
// Save / Load
// -----------------------------

func (db *MovieDatabase) Save(filename string) error {
	db.LastUpdated = time.Now()
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (db *MovieDatabase) Load(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, db)
}

// -----------------------------
// Mock TMDB Client (for pipeline testing)
// -----------------------------

type MockTMDBClient struct {
	genres []string
}

func NewMockTMDBClient() *MockTMDBClient {
	return &MockTMDBClient{}
}

func (m *MockTMDBClient) loadGenres() error {
	// Populate a set of common genres
	m.genres = []string{
		"Action", "Comedy", "Drama", "Horror", "Sci-Fi", "Romance", "Thriller",
		"Animation", "Adventure", "Family", "Fantasy", "Mystery", "Documentary",
	}
	return nil
}

func (m *MockTMDBClient) searchMovies(query string, count int) ([]MovieInfo, error) {
	// Generate synthetic but deterministic movie entries for a given query.
	results := make([]MovieInfo, 0, count)
	baseYear := 1990
	if strings.Contains(strings.ToLower(query), "202") { // recent year queries like 2021, 2022
		baseYear = 2018
	}
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("%s-%02d", strings.ReplaceAll(strings.ToLower(query), " ", "_"), i+1)
		title := fmt.Sprintf("%s Movie %02d", strings.Title(query), i+1)
		genres := []string{}
		// pick up to 2 genres deterministically from the query and i
		if i%3 == 0 {
			genres = append(genres, "Action")
		}
		if i%5 == 0 {
			genres = append(genres, "Comedy")
		}
		if len(genres) == 0 {
			genres = append(genres, m.genres[(i+len(query))%len(m.genres)])
		}
		// director cycle
		director := fmt.Sprintf("Director %d", (i%12)+1)
		year := baseYear + (i % 25) // years between baseYear..baseYear+24
		rating := 5.0 + float64((i%50))/10.0 // 5.0 .. 9.9 range
		results = append(results, MovieInfo{
			ID:          id,
			Title:       title,
			Year:        year,
			Description: fmt.Sprintf("Synthetic description for %s (query=%s)", title, query),
			Genres:      genres,
			Director:    director,
			Rating:      rating,
			Source:      "MOCK_TMDB",
			LastUpdated: time.Now().Format(time.RFC3339),
		})
	}
	return results, nil
}

// -----------------------------
// Build pipeline: collects many movies using the Mock client
// -----------------------------

func buildMovieDatabaseMock() (*MovieDatabase, error) {
	db := NewMovieDatabase()
	client := NewMockTMDBClient()

	fmt.Println("Loading genres...")
	if err := client.loadGenres(); err != nil {
		return nil, err
	}

	queries := []string{
		"action", "comedy", "drama", "horror", "sci-fi", "romance",
		"thriller", "fantasy", "animation", "documentary", "classic",
		"2023", "2022", "2021", "superhero",
	}

	targetPerQuery := 10 // aim 10+ per category (will produce >100 total)
	fmt.Println("Collecting movies for queries:")
	totalAdded := 0
	for i, q := range queries {
		fmt.Printf("[%d/%d] Searching '%s'...\n", i+1, len(queries), q)
		movies, err := client.searchMovies(q, targetPerQuery)
		if err != nil {
			fmt.Printf("  error searching %s: %v\n", q, err)
			continue
		}
		added := 0
		for _, mi := range movies {
			// Avoid duplicates based on ID
			if _, exists := db.Movies[mi.ID]; !exists {
				if err := db.Add(mi); err == nil {
					added++
					totalAdded++
				}
			}
		}
		fmt.Printf("  Added %d movies for '%s' (found %d)\n", added, q, len(movies))
		// rate-limit for politeness even though it's mock
		time.Sleep(200 * time.Millisecond)
	}
	db.LastUpdated = time.Now()
	fmt.Printf("\nCollection finished: total %d movies added\n", totalAdded)
	return db, nil
}

// -----------------------------
// Statistics helpers
// -----------------------------

func (db *MovieDatabase) PrintStatistics() {
	fmt.Println("\n=== Movie Database Statistics ===")
	fmt.Printf("Total Movies: %d\n", db.TotalCount)
	fmt.Printf("Distinct Genres: %d\n", len(db.Genres))
	fmt.Printf("Distinct Directors: %d\n", len(db.Directors))
	fmt.Printf("Years covered: %d\n", len(db.Years))
	fmt.Printf("Last Updated: %s\n\n", db.LastUpdated.Format(time.RFC3339))

	// Top genres
	type kv struct {
		Key   string
		Count int
	}
	genreList := []kv{}
	for g, ids := range db.Genres {
		genreList = append(genreList, kv{g, len(ids)})
	}
	sort.Slice(genreList, func(i, j int) bool { return genreList[i].Count > genreList[j].Count })
	fmt.Println("Top genres:")
	for i, g := range genreList {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s — %d movies\n", i+1, g.Key, g.Count)
	}

	// Rating distribution buckets (0-2,2-4,4-6,6-8,8-10)
	buckets := map[string]int{
		"0-2":   0,
		"2-4":   0,
		"4-6":   0,
		"6-8":   0,
		"8-10":  0,
	}
	for _, m := range db.Movies {
		switch {
		case m.Rating < 2:
			buckets["0-2"]++
		case m.Rating < 4:
			buckets["2-4"]++
		case m.Rating < 6:
			buckets["4-6"]++
		case m.Rating < 8:
			buckets["6-8"]++
		default:
			buckets["8-10"]++
		}
	}
	fmt.Println("\nRating distribution:")
	for k, v := range buckets {
		fmt.Printf("  %s: %d\n", k, v)
	}

	// Movies by decade
	decade := map[int]int{}
	for year := range db.Years {
		d := (year / 10) * 10
		decade[d] += len(db.Years[year])
	}
	decades := []int{}
	for d := range decade {
		decades = append(decades, d)
	}
	sort.Ints(decades)
	fmt.Println("\nMovies by decade:")
	for _, d := range decades {
		fmt.Printf("  %ds: %d\n", d, decade[d])
	}

	// Top directors
	dirList := []kv{}
	for d, ids := range db.Directors {
		dirList = append(dirList, kv{d, len(ids)})
	}
	sort.Slice(dirList, func(i, j int) bool { return dirList[i].Count > dirList[j].Count })
	fmt.Println("\nTop directors:")
	for i, d := range dirList {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s — %d movies\n", i+1, d.Key, d.Count)
	}
}

// -----------------------------
// Main
// -----------------------------

func main() {
	fmt.Println("=== Movie Database Builder (Mock TMDB) ===")

	db, err := buildMovieDatabaseMock()
	if err != nil {
		fmt.Printf("Error building database: %v\n", err)
		return
	}

	// Print some sample search results
	fmt.Println("\n--- Sample searches ---")
	sr, _ := db.Search("action")
	fmt.Printf("Search 'action' returned %d results (showing up to 3):\n", len(sr))
	for i := 0; i < len(sr) && i < 3; i++ {
		fmt.Printf("  %d. %s (%d) — Director: %s — Rating: %.1f\n", i+1, sr[i].Title, sr[i].Year, sr[i].Director, sr[i].Rating)
	}

	// Print statistics and save DB
	db.PrintStatistics()

	filename := "movie_database.json"
	if err := db.Save(filename); err != nil {
		fmt.Printf("Error saving DB: %v\n", err)
		return
	}
	info, _ := os.Stat(filename)
	fmt.Printf("\n✓ Saved %d movies to %s (size: %d KB)\n", db.TotalCount, filename, info.Size()/1024)
}

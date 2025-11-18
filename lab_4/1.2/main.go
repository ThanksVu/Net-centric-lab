package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "time"
)

const TMDBBaseURL = "https://api.themoviedb.org/3"

type TMDBGenreListResponse struct {
    Genres []struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    } `json:"genres"`
}

type TMDBSearchResponse struct {
    Page         int `json:"page"`
    Results      []struct {
        ID          int     `json:"id"`
        Title       string  `json:"title"`
        Overview    string  `json:"overview"`
        ReleaseDate string  `json:"release_date"`
        VoteAverage float64 `json:"vote_average"`
        GenreIDs    []int   `json:"genre_ids"`
        PosterPath  string  `json:"poster_path"`
    } `json:"results"`
    TotalResults int `json:"total_results"`
}

type Movie struct {
    ID          int      `json:"id"`
    Title       string   `json:"title"`
    Overview    string   `json:"overview"`
    ReleaseDate string   `json:"release_date"`
    Rating      float64  `json:"rating"`
    Genres      []string `json:"genres"`
    PosterURL   string   `json:"poster_url"`
}

type TMDBClient struct {
    APIKey     string
    BaseURL    string
    HTTPClient *http.Client
    GenreMap   map[int]string
}

func NewTMDBClient(apiKey string) *TMDBClient {
    return &TMDBClient{
        APIKey:  apiKey,
        BaseURL: TMDBBaseURL,
        HTTPClient: &http.Client{
            Timeout: 15 * time.Second,
        },
        GenreMap: make(map[int]string),
    }
} 

func (c *TMDBClient) loadGenres() error {
	endpoint := fmt.Sprintf("%s/genre/movie/list?api_key=%s", c.BaseURL, url.QueryEscape(c.APIKey))
 	resp, err := c.HTTPClient.Get(endpoint)

    if err != nil {
        return fmt.Errorf("failed to fetch genres: %v", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code %d when fetching genres", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read genres response body: %v", err)
    }

    var data TMDBGenreListResponse
    if err := json.Unmarshal(body, &data); err != nil {
        return fmt.Errorf("failed to unmarshal genres JSON: %v", err)
    }

    for _, g := range data.Genres {
        c.GenreMap[g.ID] = g.Name
    }

    fmt.Printf("Loaded %d genres\n", len(c.GenreMap))
    return nil
}

func (c *TMDBClient) searchMovies(query string, limit int) ([]Movie, error) {
    escapedQuery := url.QueryEscape(query)
    endpoint := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s", c.BaseURL, url.QueryEscape(c.APIKey), escapedQuery)
    resp, err := c.HTTPClient.Get(endpoint)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch movie search: %v", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code %d when searching movies", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read search response body: %v", err)
    }

    var sr TMDBSearchResponse
    if err := json.Unmarshal(body, &sr); err != nil {
        return nil, fmt.Errorf("failed to unmarshal search JSON: %v", err)
    }

    movies := make([]Movie, 0, len(sr.Results))
    for i, r := range sr.Results {
        if limit > 0 && i >= limit {
            break
        }
        genres := []string{}
        for _, gid := range r.GenreIDs {
            if name, ok := c.GenreMap[gid]; ok {
                genres = append(genres, name)
            }
        }
        posterURL := ""
        if r.PosterPath != "" {
            posterURL = "https://image.tmdb.org/t/p/w500/" + r.PosterPath
        }
        movies = append(movies, Movie{
            ID:          r.ID,
            Title:       r.Title,
            Overview:    r.Overview,
            ReleaseDate: r.ReleaseDate,
            Rating:      r.VoteAverage,
            Genres:      genres,
            PosterURL:   posterURL,
        })
    }

    fmt.Printf("Found %d movies\n", len(movies))
    return movies, nil
}

func saveMoviesToJSON(movies []Movie, filename string) error {
    data, err := json.MarshalIndent(movies, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal movies to JSON: %v", err)
    }
    if err := os.WriteFile(filename, data, 0644); err != nil {
        return fmt.Errorf("failed to write JSON file: %v", err)
    }
    fmt.Printf("\nSaved %d movies to %s\n", len(movies), filename)
    return nil
}

func main() {
    apiKey := "2489c0d66b08a255ae54236695bdec0e" // ← Replace this with your actual TMDB API key
    client := NewTMDBClient(apiKey)

    fmt.Println("Loading movie genres...")
    if err := client.loadGenres(); err != nil {
        fmt.Printf("Error loading genres: %v\n", err)
        return
    }

    query := "inception"
    fmt.Printf("\nSearching for: %s\n", query)
    movies, err := client.searchMovies(query, 20)
    if err != nil {
        fmt.Printf("Error searching movies: %v\n", err)
        return
    }

    // Display first two movies
    for i, m := range movies {
        if i >= 2 {
            break
        }
        fmt.Printf("\nMovie %d:\n", i+1)
        fmt.Printf("  ID: %d\n", m.ID)
        fmt.Printf("  Title: %s\n", m.Title)
        fmt.Printf("  Release Date: %s\n", m.ReleaseDate)
        fmt.Printf("  Rating: %.1f/10\n", m.Rating)
        fmt.Printf("  Genres: %v\n", m.Genres)
        fmt.Printf("  Overview: %s\n", m.Overview)
    }

    if err := saveMoviesToJSON(movies, "tmdb_results.json"); err != nil {
        fmt.Printf("Error saving JSON: %v\n", err)
    }
}

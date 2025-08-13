package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type WeatherResponse struct {
	ShortForecast string `json:"short_forecast"`
	TempCategory  string `json:"temp_category"`
}

func main() {
	http.HandleFunc("/weather", weatherHandler)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	if latStr == "" || lonStr == "" {
		http.Error(w, "Missing lat or lon parameters", http.StatusBadRequest)
		return
	}

	// Validate latitude and longitude
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		http.Error(w, "Invalid latitude", http.StatusBadRequest)
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		http.Error(w, "Invalid longitude", http.StatusBadRequest)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Get point metadata
	pointsURL := fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", lat, lon)
	req, err := http.NewRequest("GET", pointsURL, nil)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "weather-service-example stevek@gmail.com")
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch point data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Point API returned %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	var pointsData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pointsData); err != nil {
		http.Error(w, "Failed to parse point data", http.StatusInternalServerError)
		return
	}

	props, ok := pointsData["properties"].(map[string]interface{})
	if !ok {
		http.Error(w, "Invalid point data", http.StatusInternalServerError)
		return
	}

	gridX, _ := props["gridX"].(float64)
	gridY, _ := props["gridY"].(float64)
	gridId, _ := props["gridId"].(string)

	// Get forecast
	forecastURL := fmt.Sprintf("https://api.weather.gov/gridpoints/%s/%d,%d/forecast", gridId, int(gridX), int(gridY))
	req2, err := http.NewRequest("GET", forecastURL, nil)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	req2.Header.Set("User-Agent", "weather-service-example stevek@gmail.com")
	resp2, err := client.Do(req2)
	if err != nil {
		http.Error(w, "Failed to fetch forecast", http.StatusInternalServerError)
		return
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Forecast API returned %d", resp2.StatusCode), http.StatusInternalServerError)
		return
	}

	var forecastData map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&forecastData); err != nil {
		http.Error(w, "Failed to parse forecast", http.StatusInternalServerError)
		return
	}

	props2, ok := forecastData["properties"].(map[string]interface{})
	if !ok {
		http.Error(w, "Invalid forecast data", http.StatusInternalServerError)
		return
	}

	periods, ok := props2["periods"].([]interface{})
	if !ok || len(periods) == 0 {
		http.Error(w, "No forecast periods available", http.StatusInternalServerError)
		return
	}

	// Find the first daytime period (assuming it's "Today" or the next day if after sunset)
	var todayPeriod map[string]interface{}
	for _, p := range periods {
		period := p.(map[string]interface{})
		if period["isDaytime"].(bool) {
			todayPeriod = period
			break
		}
	}

	if todayPeriod == nil {
		// Fallback to first period if no daytime found, this is unlikely
		todayPeriod = periods[0].(map[string]interface{})
	}
	shortForecast, _ := todayPeriod["shortForecast"].(string)
	temp, _ := todayPeriod["temperature"].(float64)

	// Thresholds: cold < 50F, moderate 50-80F, hot > 80F; temperature in F
	var category string
	if temp < 50 {
		category = "cold"
	} else if temp <= 80 {
		category = "moderate"
	} else {
		category = "hot"
	}

	response := WeatherResponse{
		ShortForecast: shortForecast,
		TempCategory:  category,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

/*
Notes:
- I Added HTTP client timeout of 10 seconds for external API calls to prevent hanging.
- Server runs on :8080 hardcoded; could be configurable.
*/

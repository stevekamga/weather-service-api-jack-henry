package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	listenAddr = ":8080"
	userAgent  = "weather-service-sample (stevekamga@gmail.com)"
)

type pointsResp struct {
	Properties struct {
		Forecast string `json:"forecast"`
	} `json:"properties"`
}

type forecastResp struct {
	Properties struct {
		Periods []period `json:"periods"`
	} `json:"properties"`
}

type period struct {
	Name            string `json:"name"`
	StartTime       string `json:"startTime"`
	IsDaytime       bool   `json:"isDaytime"`
	Temperature     int    `json:"temperature"`
	TemperatureUnit string `json:"temperatureUnit"`
	ShortForecast   string `json:"shortForecast"`
}

type apiResponse struct {
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	PeriodName      string  `json:"periodName"`
	ShortForecast   string  `json:"shortForecast"`
	Temperature     int     `json:"temperature"`
	TemperatureUnit string  `json:"temperatureUnit"`
	TemperatureType string  `json:"temperatureType"`
	FetchedAt       string  `json:"fetchedAt"`
	Source          string  `json:"source"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/weather", handleWeather)

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", listenAddr)
	log.Fatal(srv.ListenAndServe())
}

func handleWeather(w http.ResponseWriter, r *http.Request) {
	lat, lon, err := parseLatLon(r)
	if err != nil {
		httpErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	pointsURL := fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", lat, lon)
	var p pointsResp
	if err := getJSON(r.Context(), pointsURL, &p); err != nil {
		httpErrorJSON(w, http.StatusBadGateway, fmt.Sprintf("failed to fetch points: %v", err))
		return
	}
	if p.Properties.Forecast == "" {
		httpErrorJSON(w, http.StatusBadGateway, "missing forecast URL in points response")
		return
	}

	var f forecastResp
	if err := getJSON(r.Context(), p.Properties.Forecast, &f); err != nil {
		httpErrorJSON(w, http.StatusBadGateway, fmt.Sprintf("failed to fetch forecast: %v", err))
		return
	}
	if len(f.Properties.Periods) == 0 {
		httpErrorJSON(w, http.StatusBadGateway, "no forecast periods available")
		return
	}

	todayPeriod, err := pickTodayDaytimePeriod(f.Properties.Periods)
	if err != nil {
		httpErrorJSON(w, http.StatusBadGateway, fmt.Sprintf("could not select today's daytime period: %v", err))
		return
	}

	resp := apiResponse{
		Lat:             lat,
		Lon:             lon,
		PeriodName:      todayPeriod.Name,
		ShortForecast:   todayPeriod.ShortForecast,
		Temperature:     todayPeriod.Temperature,
		TemperatureUnit: todayPeriod.TemperatureUnit,
		TemperatureType: characterizeTemp(todayPeriod.TemperatureUnit, todayPeriod.Temperature),
		FetchedAt:       time.Now().Format(time.RFC3339),
		Source:          "api.weather.gov",
	}

	writeJSON(w, http.StatusOK, resp)
}

func parseLatLon(r *http.Request) (float64, float64, error) {
	q := r.URL.Query()
	latStr := strings.TrimSpace(q.Get("lat"))
	lonStr := strings.TrimSpace(q.Get("lon"))
	if latStr == "" || lonStr == "" {
		return 0, 0, errors.New("missing lat and/or lon query parameters")
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lat: %v", err)
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lon: %v", err)
	}

	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, errors.New("lat must be -90..90, lon -180..180")
	}

	// NWS supports max 4 decimal places
	lat = math.Round(lat*10000) / 10000
	lon = math.Round(lon*10000) / 10000

	return lat, lon, nil
}

func pickTodayDaytimePeriod(periods []period) (period, error) {
	now := time.Now()

	// Try to find best daytime period for "today" or very soon
	var best period
	var bestTime time.Time
	found := false

	for _, p := range periods {
		if !p.IsDaytime {
			continue
		}
		st, err := time.Parse(time.RFC3339, p.StartTime)
		if err != nil {
			continue
		}
		if st.Before(now) {
			continue // skip past periods
		}

		// Prefer earliest future daytime period
		if !found || st.Before(bestTime) {
			best = p
			bestTime = st
			found = true
		}
	}

	if found {
		return best, nil
	}

	// Fallback: first daytime period (even if past)
	for _, p := range periods {
		if p.IsDaytime {
			return p, nil
		}
	}

	return period{}, errors.New("no daytime period found in forecast")
}

func characterizeTemp(unit string, temp int) string {
	u := strings.ToUpper(strings.TrimSpace(unit))
	switch u {
	case "F":
		if temp <= 49 {
			return "cold"
		}
		if temp >= 80 {
			return "hot"
		}
		return "moderate"
	case "C":
		if temp <= 9 {
			return "cold"
		}
		if temp >= 27 {
			return "hot"
		}
		return "moderate"
	default:
		return "moderate"
	}
}

func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/geo+json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpErrorJSON(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
